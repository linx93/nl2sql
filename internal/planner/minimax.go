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
	defaultMiniMaxModel = "MiniMax-M2.7-highspeed"
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
		Temperature: 0,
		System:      buildMiniMaxSystemPrompt(),
		Messages: []miniMaxMessage{
			{
				Role: "user",
				Content: []miniMaxContentBlock{
					{
						Type: "text",
						Text: fmt.Sprintf("domain: %s\nquestion: %s", domainID, query),
					},
				},
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
		return domain.RawPlan{}, fmt.Errorf("minimax request failed: status %d", resp.StatusCode)
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
		"Return RawPlan JSON only.",
		"Do not return SQL.",
	}, " ")
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

	return text, nil
}

// miniMaxMessagesRequest 描述发往 Anthropic 兼容接口的消息请求。
type miniMaxMessagesRequest struct {
	Model       string                `json:"model"`
	MaxTokens   int                   `json:"max_tokens"`
	Temperature int                   `json:"temperature"`
	System      string                `json:"system"`
	Messages    []miniMaxMessage      `json:"messages"`
}

// miniMaxMessage 描述一条 Anthropic 兼容消息。
type miniMaxMessage struct {
	Role    string                `json:"role"`
	Content []miniMaxContentBlock `json:"content"`
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

