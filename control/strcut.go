package control

//LoadBalancingConfig is a standardized model
// 内部使用的balancing
type LoadBalancingConfig struct {
	Strategy     string
	Filters      []string		// 指定的instance过滤方法
	RetryEnabled bool			// 是否自动重试
	RetryOnSame  int
	RetryOnNext  int
	BackOffKind  string
	BackOffMin   int
	BackOffMax   int

	SessionTimeoutInSeconds int
	SuccessiveFailedTimes   int
}

//RateLimitingConfig is a standardized model
type RateLimitingConfig struct {
	Key     string
	Enabled bool
	Rate    int
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
