package catalog

import (
	"fmt"
	"strings"
)

// Validate 校验领域语义配置是否引用了存在且允许的物理对象。
func Validate(c Catalog) error {
	for _, metric := range c.Metrics {
		if !c.HasColumn(metric.BaseTable, metric.TimeField) {
			return fmt.Errorf("unknown column %s.%s", metric.BaseTable, metric.TimeField)
		}
	}

	for _, dimension := range c.Dimensions {
		if !c.HasColumn(dimension.Table, dimension.Column) {
			return fmt.Errorf("unknown column %s.%s", dimension.Table, dimension.Column)
		}
	}

	for _, detailView := range c.DetailViews {
		if _, ok := c.TablesByName[detailView.BaseTable]; !ok {
			return fmt.Errorf("unknown table %s", detailView.BaseTable)
		}
		if err := validateFieldRefs(c, detailView.AllowedSelectColumns); err != nil {
			return err
		}
		if err := validateFieldRefs(c, detailView.AllowedFilterFields); err != nil {
			return err
		}
		if err := validateFieldRefs(c, detailView.DefaultSelectColumns); err != nil {
			return err
		}
		if err := validateFieldRefs(c, []string{detailView.RequiredTimeField, detailView.DefaultSort.Field}); err != nil {
			return err
		}
	}

	for _, role := range c.Roles {
		for _, detailViewID := range role.AllowedDetailViewIDs {
			if _, ok := c.DetailViews[detailViewID]; !ok {
				return fmt.Errorf("unknown detail view %s", detailViewID)
			}
		}
	}

	for domainID, aliases := range c.AliasesByDomain {
		if err := validateAliasTargets(c, domainID, aliases); err != nil {
			return err
		}
	}

	return nil
}

// HasColumn 判断某个表中是否存在指定列，供语义校验复用。
func (c Catalog) HasColumn(table string, column string) bool {
	columns, ok := c.ColumnsByTable[table]
	if !ok {
		return false
	}

	_, ok = columns[column]
	return ok
}

func validateFieldRefs(c Catalog, fields []string) error {
	for _, field := range fields {
		if field == "" {
			continue
		}

		table, column, ok := splitFieldRef(field)
		if !ok {
			return fmt.Errorf("invalid field reference %s", field)
		}
		if !c.HasColumn(table, column) {
			return fmt.Errorf("unknown column %s.%s", table, column)
		}
	}

	return nil
}

func validateAliasTargets(c Catalog, domainID string, aliases AliasSet) error {
	for _, targetID := range aliases.Metrics {
		metric, ok := c.Metrics[targetID]
		if !ok || metric.DomainID != domainID {
			return fmt.Errorf("unknown metric alias target %s", targetID)
		}
	}
	for _, targetID := range aliases.Dimensions {
		dimension, ok := c.Dimensions[targetID]
		if !ok || dimension.DomainID != domainID {
			return fmt.Errorf("unknown dimension alias target %s", targetID)
		}
	}
	for _, targetID := range aliases.DetailViews {
		detailView, ok := c.DetailViews[targetID]
		if !ok || detailView.DomainID != domainID {
			return fmt.Errorf("unknown detail view alias target %s", targetID)
		}
	}

	return nil
}

func splitFieldRef(field string) (string, string, bool) {
	table, column, ok := strings.Cut(field, ".")
	if !ok || table == "" || column == "" {
		return "", "", false
	}

	return table, column, true
}
