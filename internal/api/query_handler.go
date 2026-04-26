package api

import (
	"context"
	"encoding/json"
	"net/http"

	"nl2sql/internal/orchestrator"
)

// QueryService 定义 HTTP 层调用的应用服务能力。
type QueryService interface {
	// Run 执行一条完整的 NL2SQL 查询请求。
	Run(rctx context.Context, req orchestrator.QueryRequest) (orchestrator.Response, error)
}

// Handler 提供 NL2SQL HTTP 接口入口。
type Handler struct {
	// service 是实际执行查询链路的应用服务。
	service QueryService
}

// queryRequestBody 表示 POST 请求体中的核心字段。
type queryRequestBody struct {
	Query string `json:"query"`
	Domain string `json:"domain"`
}

// queryResponseBody 表示统一的成功响应结构。
type queryResponseBody struct {
	RequestID string                 `json:"request_id"`
	Status    string                 `json:"status"`
	Data      any                    `json:"data"`
	Meta      map[string]any         `json:"meta"`
	Error     map[string]string      `json:"error,omitempty"`
}

// NewHandler 创建一个 NL2SQL HTTP Handler。
func NewHandler(service QueryService) *Handler {
	return &Handler{service: service}
}

// ServeHTTP 根据方法和路径分发请求。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/api/v1/nl2sql/queries" {
		h.handlePostQueries(w, r)
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) handlePostQueries(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var body queryRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "请求体不是合法 JSON")
		return
	}

	resp, err := h.service.Run(r.Context(), orchestrator.QueryRequest{
		Query:  body.Query,
		Domain: body.Domain,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "查询执行失败")
		return
	}

	writeJSON(w, http.StatusOK, queryResponseBody{
		RequestID: resp.RequestID,
		Status:    "success",
		Data:      resp.Data,
		Meta: map[string]any{
			"query_mode":  resp.Meta.QueryMode,
			"result_kind": resp.Meta.ResultKind,
			"row_count":   resp.Meta.RowCount,
			"truncated":   resp.Meta.Truncated,
		},
	})
}

func writeError(w http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(w, statusCode, queryResponseBody{
		Status: "error",
		Error: map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
