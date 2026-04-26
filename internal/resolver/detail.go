package resolver

import (
	"fmt"
	"strings"
	"time"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	pkgclock "nl2sql/pkg/clock"
)

// ResolveDetail 将原始明细计划解析为受控明细视图上的规范化计划。
func ResolveDetail(raw domain.RawPlan, cat catalog.Catalog, role catalog.RolePolicy, clk pkgclock.Clock) (domain.ResolvedPlan, error) {
	domainSpec, aliases, err := primaryDomain(cat)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}
	if raw.QueryMode != string(domain.QueryModeDetailList) {
		return domain.ResolvedPlan{}, fmt.Errorf("unsupported query mode %s", raw.QueryMode)
	}
	if err := ensureRoleAllowsQueryMode(role, domain.QueryModeDetailList); err != nil {
		return domain.ResolvedPlan{}, err
	}

	detailViewID, ok := aliases.DetailViews[raw.DetailSubject]
	if !ok {
		return domain.ResolvedPlan{}, fmt.Errorf("detail view not allowed: %s", raw.DetailSubject)
	}

	detailView, ok := cat.DetailViews[detailViewID]
	if !ok {
		return domain.ResolvedPlan{}, fmt.Errorf("detail view not found: %s", detailViewID)
	}
	if !roleAllowsDetailView(role, detailViewID) {
		return domain.ResolvedPlan{}, fmt.Errorf("permission denied for detail view %s", detailViewID)
	}

	if detailView.RequireNarrowingFilter && len(raw.Filters) == 0 {
		return domain.ResolvedPlan{}, fmt.Errorf("detail query requires narrowing filter")
	}

	timeRange, err := resolveTimeRange(raw.TimeRange, clk)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}
	filters, err := resolveFilters(raw.Filters, detailView.AllowedFilterFields)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}

	return domain.ResolvedPlan{
		QueryMode:       domain.QueryModeDetailList,
		DetailViewID:    detailViewID,
		SelectColumnIDs: defaultSelectColumns(detailView, raw.SelectFields),
		Filters:         filters,
		TimeRange:       timeRange,
		Limit:           clampLimit(raw.Limit, effectiveDetailLimit(role.MaxLimit, detailView.MaxLimit)),
		DatasourceID:    domainSpec.DatasourceID,
	}, nil
}

func roleAllowsDetailView(role catalog.RolePolicy, detailViewID string) bool {
	for _, allowedID := range role.AllowedDetailViewIDs {
		if allowedID == detailViewID {
			return true
		}
	}

	return false
}

func defaultSelectColumns(detailView catalog.DetailViewSpec, requested []string) []string {
	if len(requested) > 0 {
		return append([]string(nil), requested...)
	}

	return append([]string(nil), detailView.DefaultSelectColumns...)
}

func resolveFilters(filters []domain.RawFilter, allowedFields []string) ([]domain.ResolvedFilter, error) {
	resolved := make([]domain.ResolvedFilter, 0, len(filters))
	fieldIndex := buildAllowedFieldIndex(allowedFields)
	for _, filter := range filters {
		fieldID, err := resolveAllowedFieldID(fieldIndex, filter.Field)
		if err != nil {
			return nil, err
		}
		operator, err := normalizeFilterOperator(filter.Operator)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, domain.ResolvedFilter{
			FieldID:  fieldID,
			Operator: operator,
			Value:    filter.Value,
		})
	}

	return resolved, nil
}

func effectiveDetailLimit(roleMaxLimit int, detailMaxLimit int) int {
	switch {
	case roleMaxLimit <= 0:
		return detailMaxLimit
	case detailMaxLimit <= 0:
		return roleMaxLimit
	case roleMaxLimit < detailMaxLimit:
		return roleMaxLimit
	default:
		return detailMaxLimit
	}
}

func resolveTimeRange(raw domain.RawTimeRange, clk pkgclock.Clock) (domain.TimeRange, error) {
	switch raw.Type {
	case "", "relative":
		result := resolveRelativeTimeRange(raw.Value, clk)
		result.Grain = raw.Grain
		return result, nil
	case "absolute":
		result, err := resolveAbsoluteTimeRange(raw.Start, raw.End)
		if err != nil {
			return domain.TimeRange{}, err
		}
		result.Grain = raw.Grain
		return result, nil
	default:
		return domain.TimeRange{}, fmt.Errorf("unsupported time range type %s", raw.Type)
	}
}

func resolveRelativeTimeRange(value string, clk pkgclock.Clock) domain.TimeRange {
	now := clk.Now()

	switch value {
	case "last_7_days":
		return domain.TimeRange{
			Start: now.AddDate(0, 0, -6),
			End:   now,
		}
	case "last_30_days", "":
		return domain.TimeRange{
			Start: now.AddDate(0, 0, -29),
			End:   now,
		}
	default:
		return domain.TimeRange{
			Start: now.AddDate(0, 0, -29),
			End:   now,
		}
	}
}

func resolveAbsoluteTimeRange(start string, end string) (domain.TimeRange, error) {
	startAt, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return domain.TimeRange{}, fmt.Errorf("invalid absolute start %s", start)
	}

	endAt, err := time.Parse(time.DateOnly, end)
	if err != nil {
		return domain.TimeRange{}, fmt.Errorf("invalid absolute end %s", end)
	}

	return domain.TimeRange{
		Start: startAt,
		End:   endAt,
	}, nil
}

func buildAllowedFieldIndex(allowedFields []string) map[string]string {
	index := make(map[string]string, len(allowedFields)*2)
	shortFieldOwners := make(map[string]string, len(allowedFields))
	ambiguousShortFields := make(map[string]struct{})

	for _, field := range allowedFields {
		index[field] = field
		_, shortField, ok := strings.Cut(field, ".")
		if !ok {
			continue
		}

		if existingOwner, exists := shortFieldOwners[shortField]; exists && existingOwner != field {
			delete(index, shortField)
			ambiguousShortFields[shortField] = struct{}{}
			continue
		}
		if _, ambiguous := ambiguousShortFields[shortField]; ambiguous {
			continue
		}

		shortFieldOwners[shortField] = field
		index[shortField] = field
	}

	return index
}

func resolveAllowedFieldID(index map[string]string, rawField string) (string, error) {
	fieldID, ok := index[rawField]
	if !ok {
		return "", fmt.Errorf("filter field not allowed: %s", rawField)
	}

	return fieldID, nil
}

func normalizeFilterOperator(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "eq", "=":
		return "=", nil
	case "ne", "!=":
		return "!=", nil
	case "gt", ">":
		return ">", nil
	case "gte", ">=":
		return ">=", nil
	case "lt", "<":
		return "<", nil
	case "lte", "<=":
		return "<=", nil
	case "like":
		return "LIKE", nil
	default:
		return "", fmt.Errorf("filter operator not allowed: %s", raw)
	}
}
