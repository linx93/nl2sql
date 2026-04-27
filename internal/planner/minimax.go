package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nl2sql/internal/domain"
)

const (
	// defaultMiniMaxBaseURL 是 MiniMax Anthropic 兼容接口的默认地址。
	defaultMiniMaxBaseURL = "https://api.minimaxi.com"
	// defaultMiniMaxModel 是仓库约定的默认规划模型。
	defaultMiniMaxModel = "MiniMax-M2.7"
)

// MiniMaxConfig 描述实时 MiniMax planner 的最小连接配置。
type MiniMaxConfig struct {
	// BaseURL 允许测试或运行时覆盖默认接口根地址。
	BaseURL string
	// APIKey 是调用 Token Plan 所需的鉴权令牌。
	APIKey string
	// Model 是本次规划请求使用的模型名称。
	Model string
	// HTTPClient 允许测试注入自定义 HTTP 客户端。
	HTTPClient *http.Client
}

// MiniMaxPlanner 负责把自然语言请求发送到 MiniMax，并解析回受控的 RawPlan。
type MiniMaxPlanner struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewMiniMaxPlanner 创建一个面向生产或测试的 MiniMax planner 客户端。
func NewMiniMaxPlanner(cfg MiniMaxConfig) MiniMaxPlanner {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultMiniMaxBaseURL
	}

	model := cfg.Model
	if model == "" {
		model = defaultMiniMaxModel
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return MiniMaxPlanner{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		model:      model,
		httpClient: httpClient,
	}
}

// Plan 调用 MiniMax 消息接口，把自然语言问题转换成仓库定义的 RawPlan。
func (p MiniMaxPlanner) Plan(ctx context.Context, query string, domainID string) (domain.RawPlan, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return domain.RawPlan{}, errors.New("minimax api key is required")
	}

	requestBody, err := json.Marshal(miniMaxMessagesRequest{
		Model:       p.model,
		MaxTokens:   1024,
		Temperature: 0.1,
		System:      buildMiniMaxSystemPrompt(),
		Messages: []miniMaxMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("domain: %s\nquestion: %s", domainID, query),
			},
		},
	})
	if err != nil {
		return domain.RawPlan{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/anthropic/v1/messages", bytes.NewReader(requestBody))
	if err != nil {
		return domain.RawPlan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return domain.RawPlan{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return domain.RawPlan{}, fmt.Errorf("minimax request failed: status %d", resp.StatusCode)
		}

		trimmedBody := strings.TrimSpace(string(body))
		if trimmedBody == "" {
			return domain.RawPlan{}, fmt.Errorf("minimax request failed: status %d", resp.StatusCode)
		}

		return domain.RawPlan{}, fmt.Errorf("minimax request failed: status %d: %s", resp.StatusCode, trimmedBody)
	}

	var payload miniMaxMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return domain.RawPlan{}, err
	}

	rawPlanJSON, err := extractTextContent(payload.Content)
	if err != nil {
		return domain.RawPlan{}, err
	}

	return DecodeRawPlanJSON([]byte(rawPlanJSON))
}

func buildMiniMaxSystemPrompt() string {
	return strings.Join([]string{
		"You are the NL2SQL planner for this repository.",
		"Return exactly one UTF-8 JSON object that matches the RawPlan contract.",
		"Do not return SQL.",
		"Do not return markdown.",
		"Do not wrap the JSON in markdown code fences.",
		"Do not include plan_id, steps, tables, fields, calculations, question_interpretation, or SQL templates.",
		`The only allowed top-level keys are "query_mode", "metrics", "dimensions", "detail_subject", "select_fields", "filters", "time_range", "order_by", "limit", and "explanation".`,
		`"query_mode" must be one of "aggregate_overview", "ranking", "trend", "detail_list".`,
		`Use this JSON shape: {"query_mode":"","metrics":[],"dimensions":[],"detail_subject":"","select_fields":[],"filters":[{"field":"","operator":"","value":""}],"time_range":{"type":"","value":"","start":"","end":"","grain":""},"order_by":[{"field":"","direction":""}],"limit":0,"explanation":""}.`,
		`Use only these operators in filters: "eq", "ne", "gt", "gte", "lt", "lte", "like".`,
		`For the ride_hailing domain, use these exact business aliases when they apply: metric "取消率", dimension "城市", detail subject "待接驾订单".`,
		`For relative time ranges, use "last_7_days" or "last_30_days". For trend grain, use "day", "week", or "month".`,
		`Example question: 最近30天取消率最高的城市`,
		`Example RawPlan: {"query_mode":"ranking","metrics":["取消率"],"dimensions":["城市"],"detail_subject":"","select_fields":[],"filters":[],"time_range":{"type":"relative","value":"last_30_days","start":"","end":"","grain":""},"order_by":[{"field":"取消率","direction":"desc"}],"limit":10,"explanation":"查询最近30天取消率最高的城市排行"}.`,
		`Example question: 最近30天取消率是多少`,
		`Example RawPlan: {"query_mode":"aggregate_overview","metrics":["取消率"],"dimensions":[],"detail_subject":"","select_fields":[],"filters":[],"time_range":{"type":"relative","value":"last_30_days","start":"","end":"","grain":""},"order_by":[],"limit":10,"explanation":"查询最近30天取消率"}.`,
		`Example question: 最近7天每天的取消率趋势`,
		`Example RawPlan: {"query_mode":"trend","metrics":["取消率"],"dimensions":[],"detail_subject":"","select_fields":[],"filters":[],"time_range":{"type":"relative","value":"last_7_days","start":"","end":"","grain":"day"},"order_by":[],"limit":10,"explanation":"查询最近7天每天的取消率趋势"}.`,
		"Return JSON only.",
	}, "\n")
}

func extractTextContent(content []miniMaxContentBlock) (string, error) {
	var builder strings.Builder
	for _, block := range content {
		if block.Type != "text" {
			continue
		}

		builder.WriteString(block.Text)
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", io.EOF
	}

	return trimMarkdownCodeFence(text), nil
}

func trimMarkdownCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return trimmed
	}

	last := len(lines) - 1
	if strings.TrimSpace(lines[last]) != "```" {
		return trimmed
	}

	return strings.TrimSpace(strings.Join(lines[1:last], "\n"))
}

// miniMaxMessagesRequest 描述发往 Anthropic 兼容接口的消息请求。
type miniMaxMessagesRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
	System      string           `json:"system"`
	Messages    []miniMaxMessage `json:"messages"`
}

// miniMaxMessage 描述一条 Anthropic 兼容消息。
type miniMaxMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// miniMaxContentBlock 描述请求或响应中的文本块。
type miniMaxContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// miniMaxMessagesResponse 描述 Anthropic 兼容消息响应。
type miniMaxMessagesResponse struct {
	Content []miniMaxContentBlock `json:"content"`
}
