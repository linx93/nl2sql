package builder

import (
	"fmt"
	"strings"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

// BuildAggregate 根据规范化聚合计划生成受控的参数化 SQL。
func BuildAggregate(plan domain.ResolvedPlan, cat catalog.Catalog) (BuildResult, error) {
	metric, err := loadAggregateMetric(plan, cat)
	if err != nil {
		return BuildResult{}, err
	}

	switch plan.QueryMode {
	case domain.QueryModeAggregateOverview:
		return buildAggregateOverview(plan, metric)
	case domain.QueryModeRanking:
		return buildAggregateRanking(plan, cat, metric)
	case domain.QueryModeTrend:
		return buildAggregateTrend(plan, metric)
	default:
		return BuildResult{}, fmt.Errorf("unsupported aggregate query mode: %s", plan.QueryMode)
	}
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

func loadAggregateMetric(plan domain.ResolvedPlan, cat catalog.Catalog) (catalog.MetricSpec, error) {
	if len(plan.MetricIDs) != 1 {
		return catalog.MetricSpec{}, fmt.Errorf("exactly one metric is required")
	}

	metric, ok := cat.Metrics[plan.MetricIDs[0]]
	if !ok {
		return catalog.MetricSpec{}, fmt.Errorf("metric not found: %s", plan.MetricIDs[0])
	}

	return metric, nil
}

func buildAggregateOverview(plan domain.ResolvedPlan, metric catalog.MetricSpec) (BuildResult, error) {
	if len(plan.DimensionIDs) != 0 {
		return BuildResult{}, fmt.Errorf("aggregate overview does not support dimensions")
	}

	metricAlias := aliasFromID(metric.ID)
	timeFieldRef := fmt.Sprintf("%s.%s", metric.BaseTable, metric.TimeField)
	sql := strings.Join([]string{
		fmt.Sprintf("SELECT %s AS %s", metric.SQLExpression, metricAlias),
		fmt.Sprintf("FROM %s", metric.BaseTable),
		fmt.Sprintf("WHERE %s BETWEEN ? AND ?", timeFieldRef),
	}, " ")

	return baseAggregateResult(plan, metric, sql, []any{plan.TimeRange.Start, plan.TimeRange.End}, []string{timeFieldRef}, 0), nil
}

func buildAggregateRanking(plan domain.ResolvedPlan, cat catalog.Catalog, metric catalog.MetricSpec) (BuildResult, error) {
	if len(plan.DimensionIDs) != 1 {
		return BuildResult{}, fmt.Errorf("exactly one dimension is required")
	}

	dimension, ok := cat.Dimensions[plan.DimensionIDs[0]]
	if !ok {
		return BuildResult{}, fmt.Errorf("dimension not found: %s", plan.DimensionIDs[0])
	}

	metricAlias := aliasFromID(metric.ID)
	dimensionAlias := aliasFromID(dimension.ID)
	dimensionRef := fmt.Sprintf("%s.%s", dimension.Table, dimension.Column)
	timeFieldRef := fmt.Sprintf("%s.%s", metric.BaseTable, metric.TimeField)

	sql := strings.Join([]string{
		fmt.Sprintf("SELECT %s AS %s,", dimensionRef, dimensionAlias),
		fmt.Sprintf("%s AS %s", metric.SQLExpression, metricAlias),
		fmt.Sprintf("FROM %s", metric.BaseTable),
		fmt.Sprintf("WHERE %s BETWEEN ? AND ?", timeFieldRef),
		fmt.Sprintf("GROUP BY %s", dimensionRef),
		fmt.Sprintf("ORDER BY %s DESC", metricAlias),
		"LIMIT ?",
	}, " ")

	return baseAggregateResult(plan, metric, sql, []any{plan.TimeRange.Start, plan.TimeRange.End, plan.Limit}, []string{dimensionRef, timeFieldRef}, plan.Limit), nil
}

func buildAggregateTrend(plan domain.ResolvedPlan, metric catalog.MetricSpec) (BuildResult, error) {
	if len(plan.DimensionIDs) != 0 {
		return BuildResult{}, fmt.Errorf("trend does not support dimensions")
	}

	timeFieldRef := fmt.Sprintf("%s.%s", metric.BaseTable, metric.TimeField)
	metricAlias := aliasFromID(metric.ID)
	timeBucketExpr, err := buildTrendBucketExpr(timeFieldRef, plan.TimeRange.Grain)
	if err != nil {
		return BuildResult{}, err
	}

	sqlParts := []string{
		fmt.Sprintf("SELECT %s AS trend_bucket,", timeBucketExpr),
		fmt.Sprintf("%s AS %s", metric.SQLExpression, metricAlias),
		fmt.Sprintf("FROM %s", metric.BaseTable),
		fmt.Sprintf("WHERE %s BETWEEN ? AND ?", timeFieldRef),
		"GROUP BY trend_bucket",
		"ORDER BY trend_bucket ASC",
	}
	args := []any{plan.TimeRange.Start, plan.TimeRange.End}
	if plan.Limit > 0 {
		sqlParts = append(sqlParts, "LIMIT ?")
		args = append(args, plan.Limit)
	}

	return baseAggregateResult(plan, metric, strings.Join(sqlParts, " "), args, []string{timeFieldRef}, plan.Limit), nil
}

func baseAggregateResult(plan domain.ResolvedPlan, metric catalog.MetricSpec, sql string, args []any, referencedCols []string, limit int) BuildResult {
	return BuildResult{
		SQL:              sql,
		Args:             args,
		ReferencedTables: []string{metric.BaseTable},
		ReferencedCols:   referencedCols,
		MetricIDs:        append([]string(nil), plan.MetricIDs...),
		DimensionIDs:     append([]string(nil), plan.DimensionIDs...),
		TimeRangeStart:   plan.TimeRange.Start,
		TimeRangeEnd:     plan.TimeRange.End,
		TimeRangeDays:    computeTimeRangeDays(plan.TimeRange),
		JoinCount:        0,
		Limit:            limit,
	}
}

func buildTrendBucketExpr(timeFieldRef string, grain string) (string, error) {
	switch grain {
	case "", "day":
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", timeFieldRef), nil
	case "week":
		return fmt.Sprintf("DATE_FORMAT(%s, '%%x-%%v')", timeFieldRef), nil
	case "month":
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m')", timeFieldRef), nil
	default:
		return "", fmt.Errorf("unsupported trend grain: %s", grain)
	}
}
