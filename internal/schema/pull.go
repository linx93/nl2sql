package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Snapshot 表示某个数据源的物理 schema 快照。
type Snapshot struct {
	// DatasourceID 标识快照对应的数据源。
	DatasourceID string `yaml:"datasource_id"`
	// Database 是数据库名称。
	Database string `yaml:"database"`
	// Tables 保存快照中的表结构。
	Tables []Table `yaml:"tables"`
}

// Table 描述快照中的一张物理表。
type Table struct {
	// Name 是物理表名。
	Name string `yaml:"name"`
	// Comment 是表中文注释。
	Comment string `yaml:"comment"`
	// Columns 保存表内列定义。
	Columns []Column `yaml:"columns"`
}

// Column 描述快照中的一列。
type Column struct {
	// Name 是列名。
	Name string `yaml:"name"`
	// DataType 是数据库列类型。
	DataType string `yaml:"data_type"`
	// Comment 是列中文注释。
	Comment string `yaml:"comment"`
}

// LoadSnapshot 从 YAML 文件装载物理 schema 快照。
func LoadSnapshot(path string) (Snapshot, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read schema snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := yaml.Unmarshal(raw, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("parse schema snapshot: %w", err)
	}

	return snapshot, nil
}

// WriteSnapshot 将物理 schema 快照写入 YAML 文件。
func WriteSnapshot(path string, snapshot Snapshot) error {
	raw, err := yaml.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal schema snapshot: %w", err)
	}

	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write schema snapshot: %w", err)
	}

	return nil
}
