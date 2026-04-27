package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"time"

	"nl2sql/internal/config"
)

// OpenFunc 抽象 datasource 打开动作，便于测试替身替换真实 sql.Open。
type OpenFunc func(driverName string, dsn string) (*sql.DB, error)

// OpenAndRegister 按运行时配置打开只读数据源，并注册到 datasource registry。
func OpenAndRegister(_ context.Context, registry *Registry, datasources map[string]config.DatasourceConfig, open OpenFunc) error {
	if registry == nil {
		return fmt.Errorf("registry is required")
	}
	if open == nil {
		open = sql.Open
	}

	ids := make([]string, 0, len(datasources))
	for id := range datasources {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		datasourceCfg := datasources[id]
		dsn := os.Getenv(datasourceCfg.DsnEnv)
		if dsn == "" {
			return fmt.Errorf("datasource %s env %s is empty", datasourceCfg.ID, datasourceCfg.DsnEnv)
		}

		db, err := open(datasourceCfg.Driver, dsn)
		if err != nil {
			return fmt.Errorf("open datasource %s: %w", datasourceCfg.ID, err)
		}

		applyPoolConfig(db, datasourceCfg)
		registry.Register(datasourceCfg.ID, db)
	}

	return nil
}

// applyPoolConfig 将配置中的连接池参数写入只读数据库连接池。
func applyPoolConfig(db *sql.DB, cfg config.DatasourceConfig) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetimeSec > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second)
	}
}

