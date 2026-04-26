package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromDir 从配置目录装载数据源定义和领域路由信息。
func LoadFromDir(root string) (RuntimeConfig, error) {
	cfg := RuntimeConfig{
		Datasources: make(map[string]DatasourceConfig),
		Domains:     make(map[string]DomainConfig),
	}

	if err := loadDatasources(filepath.Join(root, "datasources.yaml"), &cfg); err != nil {
		return RuntimeConfig{}, err
	}
	if err := loadDomains(filepath.Join(root, "domains"), &cfg); err != nil {
		return RuntimeConfig{}, err
	}

	return cfg, nil
}

type datasourcesFile struct {
	Datasources []DatasourceConfig `yaml:"datasources"`
}

func loadDatasources(path string, cfg *RuntimeConfig) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read datasources config: %w", err)
	}

	var payload datasourcesFile
	if err := yaml.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse datasources config: %w", err)
	}

	for _, datasource := range payload.Datasources {
		cfg.Datasources[datasource.ID] = datasource
	}

	return nil
}

func loadDomains(root string, cfg *RuntimeConfig) error {
	pattern := filepath.Join(root, "*", "domain.yaml")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob domain config: %w", err)
	}

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read domain config %s: %w", path, err)
		}

		var domain DomainConfig
		if err := yaml.Unmarshal(raw, &domain); err != nil {
			return fmt.Errorf("parse domain config %s: %w", path, err)
		}

		cfg.Domains[domain.ID] = domain
	}

	return nil
}
