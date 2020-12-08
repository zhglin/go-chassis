package control

//LoadBalancingConfig is a standardized model
// 内部使用的balancing
type LoadBalancingConfig struct {
	Strategy     string
	Filters      []string // 指定的instance过滤方法
	RetryEnabled bool     // 是否自动重试
	RetryOnSame  int      // 同节点的重试次数
	RetryOnNext  int      // 重试的节点数
	BackOffKind  string   // 重试策略
	BackOffMin   int      // 指数退避算法的最小时间间隔
	BackOffMax   int      // 指数退避算法的最大时间间隔

	SessionTimeoutInSeconds int
	SuccessiveFailedTimes   int
}

//RateLimitingConfig is a standardized model
// 当前invocation的限流配置
type RateLimitingConfig struct {
	Key     string // 匹配到的key
	Enabled bool   // 是否开启
	Rate    int    // 流速
}

//EgressConfig is a standardized model
type EgressConfig struct {
	Hosts []string
	Ports []*EgressPort
}

//EgressPort protocol and the corresponding port
type EgressPort struct {
	Port     int32
	Protocol string
}
