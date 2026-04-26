package builder

import "time"

// BuildResult 表示 SQL 构建器输出的受控查询结果。
type BuildResult struct {
	// SQL 是待执行的参数化 SQL。
	SQL string
	// Args 是按顺序绑定到 SQL 占位符上的参数。
	Args []any
	// ReferencedTables 保存本次 SQL 触达的表名集合。
	ReferencedTables []string
	// ReferencedCols 保存本次 SQL 触达的列名集合，使用 table.column 形式。
	ReferencedCols []string
	// MetricIDs 保存本次构建使用的指标 ID。
	MetricIDs []string
	// DimensionIDs 保存本次构建使用的维度 ID。
	DimensionIDs []string
	// TimeRangeStart 是本次 SQL 使用的时间范围起点。
	TimeRangeStart time.Time
	// TimeRangeEnd 是本次 SQL 使用的时间范围终点。
	TimeRangeEnd time.Time
	// TimeRangeDays 是时间跨度天数，用于 guard 做复杂度校验。
	TimeRangeDays int
	// JoinCount 是本次 SQL 使用的 join 数量。
	JoinCount int
	// Limit 是最终生效的返回上限。
	Limit int
}
