package builder

import (
	"fmt"
	"strings"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
)

// BuildDetail 根据规范化明细计划生成受控的参数化 SQL。
func BuildDetail(plan domain.ResolvedPlan, cat catalog.Catalog) (BuildResult, error) {
	detailView, ok := cat.DetailViews[plan.DetailViewID]
	if !ok {
		return BuildResult{}, fmt.Errorf("detail view not found: %s", plan.DetailViewID)
	}

	if err := ensureAllowedColumns(detailView, plan.SelectColumnIDs); err != nil {
		return BuildResult{}, err
	}

	selectColumns := plan.SelectColumnIDs
	if len(selectColumns) == 0 {
		selectColumns = detailView.DefaultSelectColumns
	}

	timeField := detailView.RequiredTimeField
	orderField := detailView.DefaultSort.Field
	orderDirection := strings.ToUpper(detailView.DefaultSort.Direction)

	joins, joinCount := buildDetailJoins(detailView, selectColumns, plan.Filters)
	args := []any{plan.TimeRange.Start, plan.TimeRange.End}
	whereClauses := []string{fmt.Sprintf("%s BETWEEN ? AND ?", timeField)}

	for _, filter := range plan.Filters {
		whereClauses = append(whereClauses, fmt.Sprintf("%s %s ?", filter.FieldID, filter.Operator))
		args = append(args, filter.Value)
	}

	args = append(args, plan.Limit)

	sqlParts := []string{
		fmt.Sprintf("SELECT %s", strings.Join(selectColumns, ", ")),
		fmt.Sprintf("FROM %s", detailView.BaseTable),
	}
	if joins != "" {
		sqlParts = append(sqlParts, joins)
	}
	sqlParts = append(sqlParts,
		fmt.Sprintf("WHERE %s", strings.Join(whereClauses, " AND ")),
		fmt.Sprintf("ORDER BY %s %s", orderField, orderDirection),
		"LIMIT ?",
	)

	return BuildResult{
		SQL:              strings.Join(sqlParts, " "),
		Args:             args,
		ReferencedTables: referencedTablesForDetail(detailView, selectColumns),
		ReferencedCols:   referencedColumnsForDetail(selectColumns, timeField, orderField, plan.Filters),
		TimeRangeStart:   plan.TimeRange.Start,
		TimeRangeEnd:     plan.TimeRange.End,
		TimeRangeDays:    computeTimeRangeDays(plan.TimeRange),
		JoinCount:        joinCount,
		Limit:            plan.Limit,
	}, nil
}

func ensureAllowedColumns(detailView catalog.DetailViewSpec, selectColumns []string) error {
	allowed := make(map[string]struct{}, len(detailView.AllowedSelectColumns))
	for _, column := range detailView.AllowedSelectColumns {
		allowed[column] = struct{}{}
	}

	for _, column := range selectColumns {
		if _, ok := allowed[column]; !ok {
			return fmt.Errorf("detail column not allowed: %s", column)
		}
	}

	return nil
}

func buildDetailJoins(detailView catalog.DetailViewSpec, selectColumns []string, filters []domain.ResolvedFilter) (string, int) {
	if !needsDriversJoin(detailView, selectColumns, filters) {
		return "", 0
	}

	return "LEFT JOIN drivers ON trip_orders.driver_id = drivers.driver_id", 1
}

func needsDriversJoin(detailView catalog.DetailViewSpec, selectColumns []string, filters []domain.ResolvedFilter) bool {
	allowedDriversJoin := false
	for _, joinTable := range detailView.AllowedJoins {
		if joinTable == "drivers" {
			allowedDriversJoin = true
			break
		}
	}
	if !allowedDriversJoin {
		return false
	}

	for _, column := range selectColumns {
		if strings.HasPrefix(column, "drivers.") {
			return true
		}
	}
	for _, filter := range filters {
		if strings.HasPrefix(filter.FieldID, "drivers.") {
			return true
		}
	}

	return false
}

func referencedTablesForDetail(detailView catalog.DetailViewSpec, selectColumns []string) []string {
	tables := []string{detailView.BaseTable}
	for _, column := range selectColumns {
		if strings.HasPrefix(column, "drivers.") {
			return append(tables, "drivers")
		}
	}

	return tables
}

func referencedColumnsForDetail(selectColumns []string, timeField string, orderField string, filters []domain.ResolvedFilter) []string {
	columns := append([]string(nil), selectColumns...)
	columns = append(columns, timeField, orderField)
	for _, filter := range filters {
		columns = append(columns, filter.FieldID)
	}

	return columns
}
