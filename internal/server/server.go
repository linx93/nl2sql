package server

import (
	"fmt"
	"net/http"

	"nl2sql/internal/api"
	"nl2sql/internal/catalog"
	"nl2sql/internal/config"
)

// NewMux 根据显式注入的 QueryService 组装 HTTP 路由，避免在路由层偷偷绑定占位服务。
func NewMux(service api.QueryService) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/api/v1/nl2sql/queries", api.NewHandler(service))
	return mux
}

// LoadAndValidateCatalog 装载并校验运行时配置，为显式服务装配提供统一入口。
func LoadAndValidateCatalog(configDir string) (catalog.Catalog, error) {
	if _, err := config.LoadFromDir(configDir); err != nil {
		return catalog.Catalog{}, fmt.Errorf("load config: %w", err)
	}

	cat, err := catalog.Load(configDir)
	if err != nil {
		return catalog.Catalog{}, fmt.Errorf("load catalog: %w", err)
	}
	if err := catalog.Validate(cat); err != nil {
		return catalog.Catalog{}, fmt.Errorf("validate catalog: %w", err)
	}

	return cat, nil
}
