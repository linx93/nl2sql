package executor

import (
	"context"
	"database/sql"
	"fmt"

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

// Query 执行参数化 SQL，并将结果扫描为结构化列和行。
func (e MySQLExecutor) Query(ctx context.Context, query string, args []any) (formatter.QueryResult, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
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
