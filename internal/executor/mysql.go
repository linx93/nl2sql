package executor

import (
	"context"
	"database/sql"

	"nl2sql/internal/formatter"
)

// MySQLExecutor 负责通过只读数据库连接执行参数化查询。
type MySQLExecutor struct {
	// db 是只读 MySQL 连接池。
	db *sql.DB
}

// NewMySQLExecutor 创建一个 MySQL 只读执行器。
func NewMySQLExecutor(db *sql.DB) MySQLExecutor {
	return MySQLExecutor{db: db}
}

// Query 在指定 datasource_id 下执行参数化 SQL，并将结果扫描为结构化列和行。
func (e MySQLExecutor) Query(ctx context.Context, _ string, query string, args []any) (formatter.QueryResult, error) {
	return executeQuery(ctx, e.db, query, args)
}

