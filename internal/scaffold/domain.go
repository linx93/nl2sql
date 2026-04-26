package scaffold

import (
	"fmt"
	"strings"

	"nl2sql/internal/schema"
)

// SchemaSnapshot 是 scaffold 读取的 schema 快照类型别名。
type SchemaSnapshot = schema.Snapshot

// Table 是 scaffold 读取的物理表类型别名。
type Table = schema.Table

// Column 是 scaffold 读取的物理列类型别名。
type Column = schema.Column

// ScaffoldDomain 根据 schema 快照生成保守默认关闭的领域语义配置文本。
func ScaffoldDomain(snapshot SchemaSnapshot, domainID string, tables []string) map[string]string {
	selected := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		selected[table] = struct{}{}
	}

	var dimensions strings.Builder
	dimensions.WriteString("dimensions:\n")
	for _, table := range snapshot.Tables {
		if _, ok := selected[table.Name]; !ok {
			continue
		}
		for _, column := range table.Columns {
			dimensions.WriteString(fmt.Sprintf("  - id: dimension.%s_%s\n", table.Name, column.Name))
			dimensions.WriteString(fmt.Sprintf("    display_name: %s\n", column.Comment))
			dimensions.WriteString(fmt.Sprintf("    table: %s\n", table.Name))
			dimensions.WriteString(fmt.Sprintf("    column: %s\n", column.Name))
			dimensions.WriteString("    enabled: false\n")
		}
	}

	return map[string]string{
		"domain.yaml": fmt.Sprintf("id: %s\ndisplay_name: %s\ndatasource_id: %s\ndefault_timezone: Asia/Shanghai\nenabled: false\n", domainID, domainID, snapshot.DatasourceID),
		"dimensions.yaml": dimensions.String(),
		"metrics.yaml":    "metrics: []\n",
		"detail_views.yaml": "detail_views: []\n",
		"roles.yaml":        "roles: []\n",
		"aliases.yaml":      "metrics: {}\ndimensions: {}\ndetail_views: {}\n",
	}
}
