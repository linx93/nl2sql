package server

import (
	"context"
	"errors"
	"log"
	"net/http"

	"nl2sql/internal/api"
	"nl2sql/internal/catalog"
	"nl2sql/internal/config"
	"nl2sql/internal/orchestrator"
)

// NewMux 创建用于服务启动和 smoke test 的 HTTP 路由集合。
func NewMux(configDir string) *http.ServeMux {
	_, err := config.LoadFromDir(configDir)
	if err != nil {
		log.Printf("load config failed: %v", err)
	}

	cat, err := catalog.Load(configDir)
	if err != nil {
		log.Printf("load catalog failed: %v", err)
	} else if err := catalog.Validate(cat); err != nil {
		log.Printf("validate catalog failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/api/v1/nl2sql/queries", api.NewHandler(noopService{}))
	return mux
}

type noopService struct{}

func (noopService) Run(_ context.Context, _ orchestrator.QueryRequest) (orchestrator.Response, error) {
	return orchestrator.Response{}, errors.New("service wiring not implemented")
}
