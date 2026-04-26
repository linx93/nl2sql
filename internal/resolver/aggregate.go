package resolver

import (
	"fmt"

	"nl2sql/internal/catalog"
	"nl2sql/internal/domain"
	pkgclock "nl2sql/pkg/clock"
)

// ResolveAggregate 将原始聚合计划解析为可执行的规范化计划。
func ResolveAggregate(raw domain.RawPlan, cat catalog.Catalog, role catalog.RolePolicy, _ pkgclock.Clock) (domain.ResolvedPlan, error) {
	domainSpec, aliases, err := primaryDomain(cat)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}

	queryMode, err := normalizeAggregateMode(raw.QueryMode)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}

	metricIDs, err := resolveMetricIDs(raw.Metrics, aliases, cat)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}
	dimensionIDs, err := resolveDimensionIDs(raw.Dimensions, aliases, cat)
	if err != nil {
		return domain.ResolvedPlan{}, err
	}

	return domain.ResolvedPlan{
		QueryMode:    queryMode,
		MetricIDs:    metricIDs,
		DimensionIDs: dimensionIDs,
		Limit:        clampLimit(raw.Limit, role.MaxLimit),
		DatasourceID: domainSpec.DatasourceID,
	}, nil
}

func primaryDomain(cat catalog.Catalog) (catalog.DomainSpec, catalog.AliasSet, error) {
	for domainID, domainSpec := range cat.Domains {
		return domainSpec, cat.AliasesByDomain[domainID], nil
	}

	return catalog.DomainSpec{}, catalog.AliasSet{}, fmt.Errorf("no domain loaded")
}

func normalizeAggregateMode(raw string) (domain.QueryMode, error) {
	switch raw {
	case string(domain.QueryModeAggregateOverview):
		return domain.QueryModeAggregateOverview, nil
	case string(domain.QueryModeRanking):
		return domain.QueryModeRanking, nil
	case string(domain.QueryModeTrend):
		return domain.QueryModeTrend, nil
	default:
		return "", fmt.Errorf("unsupported query mode %s", raw)
	}
}

func resolveMetricIDs(names []string, aliases catalog.AliasSet, cat catalog.Catalog) ([]string, error) {
	metricIDs := make([]string, 0, len(names))
	for _, name := range names {
		metricID, ok := aliases.Metrics[name]
		if !ok {
			return nil, fmt.Errorf("unknown metric %s", name)
		}
		if _, ok := cat.Metrics[metricID]; !ok {
			return nil, fmt.Errorf("unknown metric %s", metricID)
		}
		metricIDs = append(metricIDs, metricID)
	}

	return metricIDs, nil
}

func resolveDimensionIDs(names []string, aliases catalog.AliasSet, cat catalog.Catalog) ([]string, error) {
	dimensionIDs := make([]string, 0, len(names))
	for _, name := range names {
		dimensionID, ok := aliases.Dimensions[name]
		if !ok {
			return nil, fmt.Errorf("unknown dimension %s", name)
		}
		if _, ok := cat.Dimensions[dimensionID]; !ok {
			return nil, fmt.Errorf("unknown dimension %s", dimensionID)
		}
		dimensionIDs = append(dimensionIDs, dimensionID)
	}

	return dimensionIDs, nil
}

func clampLimit(requested int, maxLimit int) int {
	switch {
	case requested <= 0:
		return maxLimit
	case requested > maxLimit:
		return maxLimit
	default:
		return requested
	}
}
