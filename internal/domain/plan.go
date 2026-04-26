package domain

import (
	"errors"
	"time"
)

// QueryMode 表示系统允许执行的受控查询模式。
type QueryMode string

const (
	// QueryModeAggregateOverview 表示聚合概览查询。
	QueryModeAggregateOverview QueryMode = "aggregate_overview"
	// QueryModeRanking 表示按维度排名的聚合查询。
	QueryModeRanking QueryMode = "ranking"
	// QueryModeTrend 表示按时间粒度查看趋势的聚合查询。
	QueryModeTrend QueryMode = "trend"
	// QueryModeDetailList 表示受控明细列表查询。
	QueryModeDetailList QueryMode = "detail_list"
)

// RawPlan 表示模型输出的原始计划，只允许出现用户侧语义词汇。
type RawPlan struct {
	// QueryMode 是模型推断出的查询模式。
	QueryMode string `json:"query_mode"`
	// Metrics 保存用户表达的指标名称或别名。
	Metrics []string `json:"metrics"`
	// Dimensions 保存用户表达的维度名称或别名。
	Dimensions []string `json:"dimensions"`
	// DetailSubject 保存明细查询主题。
	DetailSubject string `json:"detail_subject"`
	// SelectFields 保存用户请求返回的明细字段。
	SelectFields []string `json:"select_fields"`
	// Filters 保存用户表达的原始过滤条件。
	Filters []RawFilter `json:"filters"`
	// TimeRange 保存用户表达的原始时间范围。
	TimeRange RawTimeRange `json:"time_range"`
	// OrderBy 保存用户表达的原始排序规则。
	OrderBy []RawOrder `json:"order_by"`
	// Limit 保存用户期望返回的行数上限。
	Limit int `json:"limit"`
	// Explanation 保存模型对计划的自然语言解释，供审计使用。
	Explanation string `json:"explanation"`
}

// RawFilter 表示模型生成的原始过滤条件。
type RawFilter struct {
	// Field 是用户词汇层的字段名。
	Field string `json:"field"`
	// Operator 是过滤操作符。
	Operator string `json:"operator"`
	// Value 是过滤值。
	Value any `json:"value"`
}

// RawTimeRange 表示模型输出的原始时间范围。
type RawTimeRange struct {
	// Type 标识绝对时间或相对时间。
	Type string `json:"type"`
	// Value 保存如 last_7_days 的相对时间表达。
	Value string `json:"value"`
	// Start 保存绝对时间范围起点。
	Start string `json:"start"`
	// End 保存绝对时间范围终点。
	End string `json:"end"`
	// Grain 保存趋势查询所需的时间粒度。
	Grain string `json:"grain"`
}

// RawOrder 表示模型输出的原始排序规则。
type RawOrder struct {
	// Field 是用户词汇层字段名。
	Field string `json:"field"`
	// Direction 是排序方向。
	Direction string `json:"direction"`
}

// ResolvedPlan 表示后端解析后的规范化可执行计划。
type ResolvedPlan struct {
	// QueryMode 是已归一化的查询模式。
	QueryMode QueryMode
	// MetricIDs 保存已解析的指标 ID。
	MetricIDs []string
	// DimensionIDs 保存已解析的维度 ID。
	DimensionIDs []string
	// DetailViewID 保存已解析的明细视图 ID。
	DetailViewID string
	// SelectColumnIDs 保存已解析的明细选择列。
	SelectColumnIDs []string
	// Filters 保存已解析的过滤条件。
	Filters []ResolvedFilter
	// TimeRange 保存已展开为绝对时间的范围。
	TimeRange TimeRange
	// Sort 保存已解析的排序规则。
	Sort []ResolvedSort
	// Limit 保存已裁剪后的返回上限。
	Limit int
	// DatasourceID 是该查询唯一允许访问的数据源。
	DatasourceID string
}

// ResolvedFilter 表示解析后的过滤条件。
type ResolvedFilter struct {
	// FieldID 是规范化后的字段 ID 或 table.column。
	FieldID string
	// Operator 是受控过滤操作符。
	Operator string
	// Value 是过滤值。
	Value any
}

// TimeRange 表示解析后的绝对时间范围。
type TimeRange struct {
	// Start 是时间范围起点。
	Start time.Time
	// End 是时间范围终点。
	End time.Time
	// Grain 是趋势分析使用的时间粒度。
	Grain string
}

// ResolvedSort 表示解析后的排序规则。
type ResolvedSort struct {
	// FieldID 是规范化排序字段。
	FieldID string
	// Direction 是受控排序方向。
	Direction string
}

// Validate 校验解析后计划是否满足最基本的执行前约束。
func (p ResolvedPlan) Validate() error {
	if p.DatasourceID == "" {
		return errors.New("datasource_id is required")
	}

	return nil
}
