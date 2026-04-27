package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"nl2sql/internal/audit"
	"nl2sql/internal/catalog"
	"nl2sql/internal/config"
	"nl2sql/internal/datasource"
	"nl2sql/internal/executor"
	"nl2sql/internal/orchestrator"
	"nl2sql/internal/planner"
	pkgclock "nl2sql/pkg/clock"
)

// OpenDBFunc 抽象数据库连接创建逻辑，便于 runtime 测试替换真实 sql.Open。
type OpenDBFunc func(driverName string, dsn string) (*sql.DB, error)

// RuntimeConfig 描述构建真实运行时所需的显式输入。
type RuntimeConfig struct {
	ConfigDir      string
	MiniMaxAPIKey  string
	MiniMaxModel   string
	MiniMaxBaseURL string
	AuditDriver    string
	AuditDSN       string
	OpenDatasource datasource.OpenFunc
	OpenAuditDB    OpenDBFunc
	Clock          pkgclock.Clock
}

// Runtime 聚合 server 启动后需要长期持有的关键依赖。
type Runtime struct {
	Config          config.RuntimeConfig
	Catalog         catalog.Catalog
	Registry        *datasource.Registry
	AuditDB         *sql.DB
	AuditRepository audit.Repository
	Planner         planner.MiniMaxPlanner
	Executor        executor.RegistryExecutor
	Service         *orchestrator.Service
	Mux             *http.ServeMux
}

// BuildRuntime 构建一个可运行的 NL2SQL server runtime 组合根。
func BuildRuntime(cfg RuntimeConfig) (Runtime, error) {
	configDir := cfg.ConfigDir
	if configDir == "" {
		configDir = "configs"
	}

	runtimeConfig, err := config.LoadFromDir(configDir)
	if err != nil {
		return Runtime{}, fmt.Errorf("load runtime config: %w", err)
	}

	cat, err := LoadAndValidateCatalog(configDir)
	if err != nil {
		return Runtime{}, err
	}

	registry := datasource.NewRegistry()
	if err := datasource.OpenAndRegister(nil, registry, runtimeConfig.Datasources, cfg.OpenDatasource); err != nil {
		return Runtime{}, fmt.Errorf("open datasources: %w", err)
	}

	auditDB, err := openAuditDB(cfg)
	if err != nil {
		return Runtime{}, err
	}
	auditRepository := audit.NewRepository(auditDB)

	plannerClient := planner.NewMiniMaxPlanner(planner.MiniMaxConfig{
		BaseURL: cfg.MiniMaxBaseURL,
		APIKey:  cfg.MiniMaxAPIKey,
		Model:   cfg.MiniMaxModel,
	})
	exec := executor.NewRegistryExecutor(registry)

	clk := cfg.Clock
	if clk == nil {
		clk = pkgclock.SystemClock{}
	}

	service := orchestrator.NewService(cat, plannerClient, exec, auditRepository, clk)
	mux := NewMux(&service)

	return Runtime{
		Config:          runtimeConfig,
		Catalog:         cat,
		Registry:        registry,
		AuditDB:         auditDB,
		AuditRepository: auditRepository,
		Planner:         plannerClient,
		Executor:        exec,
		Service:         &service,
		Mux:             mux,
	}, nil
}

func openAuditDB(cfg RuntimeConfig) (*sql.DB, error) {
	auditDSN := cfg.AuditDSN
	if auditDSN == "" {
		return nil, fmt.Errorf("audit dsn is required")
	}

	driver := cfg.AuditDriver
	if driver == "" {
		driver = "mysql"
	}

	open := cfg.OpenAuditDB
	if open == nil {
		open = sql.Open
	}

	db, err := open(driver, auditDSN)
	if err != nil {
		return nil, fmt.Errorf("open audit db: %w", err)
	}

	return db, nil
}

