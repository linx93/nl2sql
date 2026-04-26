package builder

import (
	"fmt"
	"strings"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

// BuildAggregate 根据规范化聚合计划生成受控的参数化 SQL。
func BuildAggregate(plan domain.ResolvedPlan, cat catalog.Catalog) (BuildResult, error) {
	if len(plan.MetricIDs) != 1 {
		return BuildResult{}, fmt.Errorf("exactly one metric is required")
	}
	if len(plan.DimensionIDs) != 1 {
		return BuildResult{}, fmt.Errorf("exactly one dimension is required")
	}

	metric, ok := cat.Metrics[plan.MetricIDs[0]]
	if !ok {
		return BuildResult{}, fmt.Errorf("metric not found: %s", plan.MetricIDs[0])
	}
	dimension, ok := cat.Dimensions[plan.DimensionIDs[0]]
	if !ok {
		return BuildResult{}, fmt.Errorf("dimension not found: %s", plan.DimensionIDs[0])
	}

	metricAlias := aliasFromID(metric.ID)
	dimensionRef := fmt.Sprintf("%s.%s", dimension.Table, dimension.Column)
	timeFieldRef := fmt.Sprintf("%s.%s", metric.BaseTable, metric.TimeField)

	sql := strings.Join([]string{
		fmt.Sprintf("SELECT %s AS %s,", dimensionRef, aliasFromID(dimension.ID)),
		fmt.Sprintf("%s AS %s", metric.SQLExpression, metricAlias),
		fmt.Sprintf("FROM %s", metric.BaseTable),
		fmt.Sprintf("WHERE %s BETWEEN ? AND ?", timeFieldRef),
		fmt.Sprintf("GROUP BY %s", dimensionRef),
		fmt.Sprintf("ORDER BY %s DESC", metricAlias),
		"LIMIT ?",
	}, " ")

	return BuildResult{
		SQL:              sql,
		Args:             []any{plan.TimeRange.Start, plan.TimeRange.End, plan.Limit},
		ReferencedTables: []string{metric.BaseTable},
		ReferencedCols: []string{
			dimensionRef,
			timeFieldRef,
		},
		MetricIDs:      append([]string(nil), plan.MetricIDs...),
		DimensionIDs:   append([]string(nil), plan.DimensionIDs...),
		TimeRangeStart: plan.TimeRange.Start,
		TimeRangeEnd:   plan.TimeRange.End,
		TimeRangeDays:  computeTimeRangeDays(plan.TimeRange),
		JoinCount:      0,
		Limit:          plan.Limit,
	}, nil
}

func aliasFromID(id string) string {
	return strings.NewReplacer(".", "_").Replace(id)
}

func computeTimeRangeDays(r domain.TimeRange) int {
	if r.Start.IsZero() || r.End.IsZero() {
		return 0
	}

	hours := r.End.Sub(r.Start).Hours()
	if hours <= 0 {
		return 0
	}

	return int(hours/24) + 1
}
