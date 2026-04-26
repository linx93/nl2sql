package config

// RuntimeConfig 表示运行期需要装载的全部基础配置。
type RuntimeConfig struct {
	// Datasources 以数据源 ID 为键，便于后续按域快速路由。
	Datasources map[string]DatasourceConfig
	// Domains 以领域 ID 为键，便于请求阶段按 domain 查找目标数据源。
	Domains map[string]DomainConfig
}

// DatasourceConfig 描述只读 MySQL 数据源及连接池参数。
type DatasourceConfig struct {
	// ID 是运行时数据源唯一标识。
	ID string `yaml:"id"`
	// Driver 指定数据库驱动类型，v1 固定为 mysql。
	Driver string `yaml:"driver"`
	// DsnEnv 是存放只读连接串的环境变量名。
	DsnEnv string `yaml:"dsn_env"`
	// Database 是目标数据库名称。
	Database string `yaml:"database"`
	// MaxOpenConns 是连接池最大打开连接数。
	MaxOpenConns int `yaml:"max_open_conns"`
	// MaxIdleConns 是连接池最大空闲连接数。
	MaxIdleConns int `yaml:"max_idle_conns"`
	// ConnMaxLifetimeSec 是连接最大存活秒数。
	ConnMaxLifetimeSec int `yaml:"conn_max_lifetime_sec"`
}

// DomainConfig 描述业务领域到数据源的静态绑定关系。
type DomainConfig struct {
	// ID 是领域唯一标识。
	ID string `yaml:"id"`
	// DisplayName 是面向业务的中文展示名称。
	DisplayName string `yaml:"display_name"`
	// DatasourceID 指向该领域必须使用的只读数据源。
	DatasourceID string `yaml:"datasource_id"`
	// DefaultTimezone 用于补齐时间范围解析的默认时区。
	DefaultTimezone string `yaml:"default_timezone"`
	// Enabled 控制该领域是否开放给运行时。
	Enabled bool `yaml:"enabled"`
}
