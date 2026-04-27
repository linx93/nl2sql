package catalog

// Catalog 表示运行时装载后的语义目录骨架。
type Catalog struct {
	// Schemas 保存按数据源组织的物理 schema 快照。
	Schemas []SchemaSnapshot
	// Domains 保存按领域 ID 索引的领域配置。
	Domains map[string]DomainSpec
	// Metrics 保存按指标 ID 索引的指标语义定义。
	Metrics map[string]MetricSpec
	// Dimensions 保存按维度 ID 索引的维度语义定义。
	Dimensions map[string]DimensionSpec
	// DetailViews 保存按明细视图 ID 索引的明细视图定义。
	DetailViews map[string]DetailViewSpec
	// Roles 保存按角色 ID 索引的角色策略定义。
	Roles map[string]RolePolicy
	// AliasesByDomain 保存各领域的自然语言别名映射。
	AliasesByDomain map[string]AliasSet
	// TablesByName 以表名索引物理表定义，供后续语义校验复用。
	TablesByName map[string]TableSpec
	// ColumnsByTable 以表名和列名索引物理列定义，供后续白名单校验复用。
	ColumnsByTable map[string]map[string]ColumnSpec
}

// SchemaSnapshot 表示从 information_schema 拉取后生成的一份只读快照。
type SchemaSnapshot struct {
	// DatasourceID 标识该快照对应的数据源。
	DatasourceID string `yaml:"datasource_id"`
	// Database 是快照所属数据库名称。
	Database string `yaml:"database"`
	// Tables 保存该数据源下可见的表清单。
	Tables []TableSpec `yaml:"tables"`
}

// TableSpec 描述单张物理表的结构信息。
type TableSpec struct {
	// Name 是物理表名。
	Name string `yaml:"name"`
	// Comment 是数据库中的中文注释。
	Comment string `yaml:"comment"`
	// Columns 保存表内列定义。
	Columns []ColumnSpec `yaml:"columns"`
}

// ColumnSpec 描述单个物理列的元信息。
type ColumnSpec struct {
	// Name 是物理列名。
	Name string `yaml:"name"`
	// DataType 是数据库列类型。
	DataType string `yaml:"data_type"`
	// Comment 是数据库中的中文注释。
	Comment string `yaml:"comment"`
}

// DomainSpec 描述业务领域到数据源的绑定关系。
type DomainSpec struct {
	// ID 是领域唯一标识。
	ID string `yaml:"id"`
	// DisplayName 是领域中文展示名。
	DisplayName string `yaml:"display_name"`
	// DatasourceID 指向运行时必须使用的只读数据源。
	DatasourceID string `yaml:"datasource_id"`
	// DefaultTimezone 指定领域默认时区。
	DefaultTimezone string `yaml:"default_timezone"`
	// Enabled 控制领域是否可被运行时使用。
	Enabled bool `yaml:"enabled"`
}

// MetricSpec 描述聚合指标的语义定义。
type MetricSpec struct {
	// ID 是指标唯一标识。
	ID string `yaml:"id"`
	// DomainID 标识该指标属于哪个领域。
	DomainID string
	// DisplayName 是指标中文名称。
	DisplayName string `yaml:"display_name"`
	// BaseTable 是指标聚合默认起始表。
	BaseTable string `yaml:"base_table"`
	// SQLExpression 是受控的指标 SQL 片段。
	SQLExpression string `yaml:"sql_expression"`
	// TimeField 是指标允许使用的时间列。
	TimeField string `yaml:"time_field"`
	// Enabled 控制指标是否可被解析器使用。
	Enabled bool `yaml:"enabled"`
}

// DimensionSpec 描述可分组维度的语义定义。
type DimensionSpec struct {
	// ID 是维度唯一标识。
	ID string `yaml:"id"`
	// DomainID 标识该维度属于哪个领域。
	DomainID string
	// DisplayName 是维度中文名称。
	DisplayName string `yaml:"display_name"`
	// Table 是维度字段所属物理表。
	Table string `yaml:"table"`
	// Column 是维度字段列名。
	Column string `yaml:"column"`
	// Enabled 控制维度是否可被解析器使用。
	Enabled bool `yaml:"enabled"`
}

// DetailViewSpec 描述受控明细查询允许暴露的视图。
type DetailViewSpec struct {
	// ID 是明细视图唯一标识。
	ID string `yaml:"id"`
	// DomainID 标识该明细视图属于哪个领域。
	DomainID string
	// DisplayName 是明细视图中文名称。
	DisplayName string `yaml:"display_name"`
	// BaseTable 是明细查询默认起始表。
	BaseTable string `yaml:"base_table"`
	// AllowedJoins 是允许使用的关联表名集合。
	AllowedJoins []string `yaml:"allowed_joins"`
	// DefaultSelectColumns 是未显式选择时返回的默认字段列表。
	DefaultSelectColumns []string `yaml:"default_select_columns"`
	// AllowedSelectColumns 是允许选择的字段白名单。
	AllowedSelectColumns []string `yaml:"allowed_select_columns"`
	// AllowedFilterFields 是允许过滤的字段白名单。
	AllowedFilterFields []string `yaml:"allowed_filter_fields"`
	// RequiredTimeField 是明细查询强制要求的时间字段。
	RequiredTimeField string `yaml:"required_time_field"`
	// PresetFilters 是命中该明细视图后必须附加的静态过滤条件。
	PresetFilters []FilterSpec `yaml:"preset_filters"`
	// DefaultSort 是默认排序规则。
	DefaultSort SortSpec `yaml:"default_sort"`
	// MaxLimit 是明细查询最大返回行数。
	MaxLimit int `yaml:"max_limit"`
	// MaxTimeRangeDays 是允许的最大时间跨度。
	MaxTimeRangeDays int `yaml:"max_time_range_days"`
	// RequireNarrowingFilter 控制是否必须带收敛过滤条件。
	RequireNarrowingFilter bool `yaml:"require_narrowing_filter"`
	// RowPolicyKey 是行权限策略键。
	RowPolicyKey string `yaml:"row_policy_key"`
	// MaskedColumns 是需要脱敏的字段集合。
	MaskedColumns []string `yaml:"masked_columns"`
	// Enabled 控制明细视图是否启用。
	Enabled bool `yaml:"enabled"`
}

// SortSpec 描述固定的字段排序规则。
type SortSpec struct {
	// Field 是排序字段，要求使用 table.column 形式。
	Field string `yaml:"field"`
	// Direction 是排序方向，如 asc 或 desc。
	Direction string `yaml:"direction"`
}

// FilterSpec 描述可配置的静态过滤条件。
type FilterSpec struct {
	// Field 是过滤字段，要求使用 table.column 形式或白名单短字段。
	Field string `yaml:"field"`
	// Operator 是过滤操作符，要求与运行时受控操作符集合一致。
	Operator string `yaml:"operator"`
	// Value 是过滤值。
	Value string `yaml:"value"`
}

// RolePolicy 描述角色可执行的查询能力边界。
type RolePolicy struct {
	// ID 是角色唯一标识。
	ID string `yaml:"id"`
	// DomainID 标识该角色属于哪个领域。
	DomainID string
	// DisplayName 是角色中文名称。
	DisplayName string `yaml:"display_name"`
	// AllowedQueryModes 列出允许使用的查询模式。
	AllowedQueryModes []string `yaml:"allowed_query_modes"`
	// AllowedDetailViewIDs 列出允许访问的明细视图。
	AllowedDetailViewIDs []string `yaml:"allowed_detail_view_ids"`
	// MaxLimit 是角色允许的通用最大 limit。
	MaxLimit int `yaml:"max_limit"`
}

// AliasSet 保存按类别划分的自然语言别名映射。
type AliasSet struct {
	// Metrics 保存指标中文别名到指标 ID 的映射。
	Metrics map[string]string `yaml:"metrics"`
	// Dimensions 保存维度中文别名到维度 ID 的映射。
	Dimensions map[string]string `yaml:"dimensions"`
	// DetailViews 保存明细主题中文别名到明细视图 ID 的映射。
	DetailViews map[string]string `yaml:"detail_views"`
	// FilterValues 保存按字段划分的过滤值别名映射，例如城市名称到城市编码。
	FilterValues map[string]map[string]string `yaml:"filter_values"`
}
