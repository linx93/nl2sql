package formatter

import (
	"fmt"

	"nl2sql/internal/domain"
)

// Column 表示响应中的一列结构化结果。
type Column struct {
	// Name 是面向程序消费的列名。
	Name string
	// Label 是面向业务用户展示的列标签。
	Label string
}

// QueryResult 表示执行后待格式化的原始结果。
type QueryResult struct {
	// QueryMode 标识本次查询属于聚合还是明细路径。
	QueryMode domain.QueryMode
	// Columns 保存返回列定义。
	Columns []Column
	// Rows 保存行数据。
	Rows [][]any
	// Summary 是可选的预先生成摘要。
	Summary string
	// Limit 是本次结果使用的截断上限。
	Limit int
	// Truncated 表示结果是否被截断。
	Truncated bool
	// TotalRows 是本次返回的结果行数。
	TotalRows int
	// ResultKind 用于区分 aggregate/detail 等结果类型。
	ResultKind string
}

// ResponseData 表示最终返回给 API 层的结构化数据。
type ResponseData struct {
	// Columns 保存列定义。
	Columns []Column
	// Rows 保存结果行。
	Rows [][]any
	// Summary 是面向用户的中文摘要。
	Summary string
	// Truncated 标识结果是否被截断。
	Truncated bool
	// ResultKind 标识结果类型。
	ResultKind string
	// RowCount 是本次返回的行数。
	RowCount int
}

// Format 将执行结果格式化为 API 友好的响应结构。
func Format(result QueryResult) ResponseData {
	summary := result.Summary
	if summary == "" {
		summary = defaultSummary(result)
	}

	return ResponseData{
		Columns:   append([]Column(nil), result.Columns...),
		Rows:      append([][]any(nil), result.Rows...),
		Summary:   summary,
		Truncated: result.Truncated,
		ResultKind: func() string {
			if result.ResultKind != "" {
				return result.ResultKind
			}
			if result.QueryMode == domain.QueryModeDetailList {
				return "detail"
			}
			return "aggregate"
		}(),
		RowCount: len(result.Rows),
	}
}

func defaultSummary(result QueryResult) string {
	if result.QueryMode == domain.QueryModeDetailList {
		if result.Truncated {
			return fmt.Sprintf("已返回前%d条明细结果。", result.Limit)
		}
		return fmt.Sprintf("共返回%d条明细结果。", len(result.Rows))
	}

	return fmt.Sprintf("共返回%d条聚合结果。", len(result.Rows))
}
