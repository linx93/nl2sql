package executor

import (
	"context"
	"database/sql"
	"fmt"

	"nl2sql/internal/formatter"
)

// datasourceRegistry 定义按 datasource_id 解析只读连接池的最小能力。
type datasourceRegistry interface {
	ForDatasource(datasourceID string) (*sql.DB, error)
}

type queryRunner func(ctx context.Context, db *sql.DB, query string, args []any) (formatter.QueryResult, error)

// RegistryExecutor 根据 ResolvedPlan 中的 datasource_id 选择只读连接池执行查询。
type RegistryExecutor struct {
	registry datasourceRegistry
	run      queryRunner
}

// NewRegistryExecutor 创建基于 datasource registry 的只读执行器。
func NewRegistryExecutor(registry datasourceRegistry) RegistryExecutor {
	return RegistryExecutor{
		registry: registry,
		run:      executeQuery,
	}
}

// Query 先解析 datasource_id，再在对应只读连接池上执行参数化 SQL。
func (e RegistryExecutor) Query(ctx context.Context, datasourceID string, query string, args []any) (formatter.QueryResult, error) {
	db, err := e.registry.ForDatasource(datasourceID)
	if err != nil {
		return formatter.QueryResult{}, err
	}

	return e.run(ctx, db, query, args)
}

// executeQuery 复用统一的 MySQL 查询扫描逻辑，避免多执行器复制业务外的样板代码。
func executeQuery(ctx context.Context, db *sql.DB, query string, args []any) (formatter.QueryResult, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return formatter.QueryResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return formatter.QueryResult{}, fmt.Errorf("read columns: %w", err)
	}

	resultColumns := make([]formatter.Column, 0, len(columns))
	for _, column := range columns {
		resultColumns = append(resultColumns, formatter.Column{
			Name:  column,
			Label: column,
		})
	}

	resultRows := make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}

		if err := rows.Scan(destinations...); err != nil {
			return formatter.QueryResult{}, fmt.Errorf("scan row: %w", err)
		}

		resultRows = append(resultRows, values)
	}

	if err := rows.Err(); err != nil {
		return formatter.QueryResult{}, fmt.Errorf("iterate rows: %w", err)
	}

	return formatter.QueryResult{
		Columns: resultColumns,
		Rows:    resultRows,
	}, nil
}

