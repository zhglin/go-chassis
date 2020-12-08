package model

/*
cse:
  loadbalance:
    TargetService:
      backoff:
        maxMs: 400
        minMs: 200
        kind: constant
      retryEnabled: false
      retryOnNext: 2
      retryOnSame: 3
      serverListFilters: zoneaware
      strategy:
        name: WeightedResponse
    backoff:
      maxMs: 400
      minMs: 200
      kind: constant
    retryEnabled: false
    retryOnNext: 2
    retryOnSame: 3
    serverListFilters: zoneaware
    strategy:
      name: WeightedResponse
*/
// LBWrapper loadbalancing structure
type LBWrapper struct {
	Prefix LoadBalancingConfig `yaml:"cse"`
}

// LoadBalancingConfig loadbalancing structure
type LoadBalancingConfig struct {
	LBConfig LoadBalancing `yaml:"loadbalance"`
}

// LoadBalancing loadbalancing structure
type LoadBalancing struct {
	Strategy              map[string]string            `yaml:"strategy"`
	RetryEnabled          bool                         `yaml:"retryEnabled"`
	RetryOnNext           int                          `yaml:"retryOnNext"`
	RetryOnSame           int                          `yaml:"retryOnSame"`
	Filters               string                       `yaml:"serverListFilters"`
	Backoff               BackoffStrategy              `yaml:"backoff"`
	SessionStickinessRule SessionStickinessRule        `yaml:"SessionStickinessRule"`
	AnyService            map[string]LoadBalancingSpec `yaml:",inline"` // 针对不同的service的配置
}

// LoadBalancingSpec loadbalancing structure
type LoadBalancingSpec struct {
	Strategy              map[string]string     `yaml:"strategy"`
	SessionStickinessRule SessionStickinessRule `yaml:"SessionStickinessRule"`
	RetryEnabled          bool                  `yaml:"retryEnabled"`
	RetryOnNext           int                   `yaml:"retryOnNext"`
	RetryOnSame           int                   `yaml:"retryOnSame"`
	Backoff               BackoffStrategy       `yaml:"backoff"` // 补偿
}

// SessionStickinessRule loadbalancing structure
type SessionStickinessRule struct {
	SessionTimeoutInSeconds int `yaml:"sessionTimeoutInSeconds"` // 会话过期时间
	SuccessiveFailedTimes   int `yaml:"successiveFailedTimes"`   //请求失败次数达到了就直接切换节点
}

// BackoffStrategy back off strategy
type BackoffStrategy struct {
	Kind  string `yaml:"kind"`
	MinMs int    `yaml:"minMs"`
	MaxMs int    `yaml:"maxMs"`
}
