package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load 从给定目录装载 generated schema，并建立按表和列访问的索引。
func Load(root string) (Catalog, error) {
	result := Catalog{
		Domains:         make(map[string]DomainSpec),
		Metrics:         make(map[string]MetricSpec),
		Dimensions:      make(map[string]DimensionSpec),
		DetailViews:     make(map[string]DetailViewSpec),
		Roles:           make(map[string]RolePolicy),
		AliasesByDomain: make(map[string]AliasSet),
		TablesByName:    make(map[string]TableSpec),
		ColumnsByTable:  make(map[string]map[string]ColumnSpec),
	}

	pattern := filepath.Join(root, "schemas", "*.generated.yaml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return Catalog{}, fmt.Errorf("glob schema files: %w", err)
	}

	for _, path := range paths {
		snapshot, err := loadSnapshot(path)
		if err != nil {
			return Catalog{}, err
		}

		result.Schemas = append(result.Schemas, snapshot)
		for _, table := range snapshot.Tables {
			result.TablesByName[table.Name] = table

			columnIndex := make(map[string]ColumnSpec)
			for _, column := range table.Columns {
				columnIndex[column.Name] = column
			}
			result.ColumnsByTable[table.Name] = columnIndex
		}
	}

	if err := loadSemanticDomains(root, &result); err != nil {
		return Catalog{}, err
	}

	return result, nil
}

func loadSnapshot(path string) (SchemaSnapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SchemaSnapshot{}, fmt.Errorf("read schema snapshot %s: %w", path, err)
	}

	var snapshot SchemaSnapshot
	if err := yaml.Unmarshal(raw, &snapshot); err != nil {
		return SchemaSnapshot{}, fmt.Errorf("parse schema snapshot %s: %w", path, err)
	}

	return snapshot, nil
}

type metricsFile struct {
	Metrics []MetricSpec `yaml:"metrics"`
}

type dimensionsFile struct {
	Dimensions []DimensionSpec `yaml:"dimensions"`
}

type detailViewsFile struct {
	DetailViews []DetailViewSpec `yaml:"detail_views"`
}

type rolesFile struct {
	Roles []RolePolicy `yaml:"roles"`
}

func loadSemanticDomains(root string, catalog *Catalog) error {
	pattern := filepath.Join(root, "domains", "*", "domain.yaml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob domain files: %w", err)
	}

	for _, domainPath := range paths {
		domainDir := filepath.Dir(domainPath)
		domain, err := loadDomain(domainPath)
		if err != nil {
			return err
		}

		catalog.Domains[domain.ID] = domain

		metrics, err := loadMetrics(filepath.Join(domainDir, "metrics.yaml"), domain.ID)
		if err != nil {
			return err
		}
		for _, metric := range metrics {
			catalog.Metrics[metric.ID] = metric
		}

		dimensions, err := loadDimensions(filepath.Join(domainDir, "dimensions.yaml"), domain.ID)
		if err != nil {
			return err
		}
		for _, dimension := range dimensions {
			catalog.Dimensions[dimension.ID] = dimension
		}

		detailViews, err := loadDetailViews(filepath.Join(domainDir, "detail_views.yaml"), domain.ID)
		if err != nil {
			return err
		}
		for _, detailView := range detailViews {
			catalog.DetailViews[detailView.ID] = detailView
		}

		roles, err := loadRoles(filepath.Join(domainDir, "roles.yaml"), domain.ID)
		if err != nil {
			return err
		}
		for _, role := range roles {
			catalog.Roles[role.ID] = role
		}

		aliases, err := loadAliases(filepath.Join(domainDir, "aliases.yaml"))
		if err != nil {
			return err
		}
		catalog.AliasesByDomain[domain.ID] = aliases
	}

	return nil
}

func loadDomain(path string) (DomainSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return DomainSpec{}, fmt.Errorf("read domain config %s: %w", path, err)
	}

	var domain DomainSpec
	if err := yaml.Unmarshal(raw, &domain); err != nil {
		return DomainSpec{}, fmt.Errorf("parse domain config %s: %w", path, err)
	}

	return domain, nil
}

func loadMetrics(path string, domainID string) ([]MetricSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metrics config %s: %w", path, err)
	}

	var payload metricsFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse metrics config %s: %w", path, err)
	}

	for i := range payload.Metrics {
		payload.Metrics[i].DomainID = domainID
	}

	return payload.Metrics, nil
}

func loadDimensions(path string, domainID string) ([]DimensionSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dimensions config %s: %w", path, err)
	}

	var payload dimensionsFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse dimensions config %s: %w", path, err)
	}

	for i := range payload.Dimensions {
		payload.Dimensions[i].DomainID = domainID
	}

	return payload.Dimensions, nil
}

func loadDetailViews(path string, domainID string) ([]DetailViewSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read detail views config %s: %w", path, err)
	}

	var payload detailViewsFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse detail views config %s: %w", path, err)
	}

	for i := range payload.DetailViews {
		payload.DetailViews[i].DomainID = domainID
	}

	return payload.DetailViews, nil
}

func loadRoles(path string, domainID string) ([]RolePolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read roles config %s: %w", path, err)
	}

	var payload rolesFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse roles config %s: %w", path, err)
	}

	for i := range payload.Roles {
		payload.Roles[i].DomainID = domainID
	}

	return payload.Roles, nil
}

func loadAliases(path string) (AliasSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AliasSet{}, fmt.Errorf("read aliases config %s: %w", path, err)
	}

	var aliases AliasSet
	if err := yaml.Unmarshal(raw, &aliases); err != nil {
		return AliasSet{}, fmt.Errorf("parse aliases config %s: %w", path, err)
	}

	return aliases, nil
}
