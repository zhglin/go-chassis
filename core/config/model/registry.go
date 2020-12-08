package model

//RegistryStruct SC information	注册中心配置
type RegistryStruct struct {
	Disable         bool                     `yaml:"disabled"` // 是否关闭
	Type            string                   `yaml:"type"`     // 注册中心类型 区分不同的注册中心
	Scope           string                   `yaml:"scope"`
	AutoDiscovery   bool                     `yaml:"autodiscovery"` // 是否刷新注册中心可用节点
	AutoIPIndex     bool                     `yaml:"autoIPIndex"`
	Address         string                   `yaml:"address"`         // ,分隔的地址
	RefreshInterval string                   `yaml:"refreshInterval"` // 定时更新依赖service的时间间隔 默认30s
	Watch           bool                     `yaml:"watch"`           // 是否watch自身节点
	AutoRegister    string                   `yaml:"register"`
	APIVersion      RegistryAPIVersionStruct `yaml:"api"`

	HealthCheck bool   `yaml:"healthCheck"` // 未发现到的节点是否进行healthCheck
	CacheIndex  bool   `yaml:"cacheIndex"`
	ConfigPath  string `yaml:"configPath"`
}

// RegistryAPIVersionStruct registry api version structure
type RegistryAPIVersionStruct struct {
	Version string `yaml:"version"` // api的版本号
}
