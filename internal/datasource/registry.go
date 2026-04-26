package datasource

import (
	"database/sql"
	"fmt"

	"nl2sql/internal/config"
)

// Registry 保存运行时可用的只读数据库连接池，并负责按领域路由。
type Registry struct {
	// pools 以数据源 ID 为键保存已注册的只读连接池。
	pools map[string]*sql.DB
}

// NewRegistry 创建空的数据源注册表。
func NewRegistry() *Registry {
	return &Registry{
		pools: make(map[string]*sql.DB),
	}
}

// Register 将一个只读连接池注册到指定数据源 ID 下。
func (r *Registry) Register(datasourceID string, db *sql.DB) {
	r.pools[datasourceID] = db
}

// ForDomain 根据领域配置返回该领域唯一允许使用的数据源连接池。
func (r *Registry) ForDomain(domainID string, domains map[string]config.DomainConfig) (*sql.DB, error) {
	domain, ok := domains[domainID]
	if !ok {
		return nil, fmt.Errorf("domain %s not found", domainID)
	}

	db, ok := r.pools[domain.DatasourceID]
	if !ok {
		return nil, fmt.Errorf("datasource %s not registered", domain.DatasourceID)
	}

	return db, nil
}
