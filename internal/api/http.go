package api

import "net/http"

// NewMux 创建包含 NL2SQL 核心路由的 HTTP 多路复用器。
func NewMux(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/nl2sql/queries", handler)
	return mux
}
