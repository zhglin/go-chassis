package servicecomb

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-archaius/event"
	"github.com/go-chassis/openlog"
)

//RegisterKeys registers a config key to the archaius
func RegisterKeys(eventListener event.Listener, keys ...string) {
	err := archaius.RegisterListener(eventListener, keys...)
	if err != nil {
		openlog.Error(err.Error())
	}
}

//Init is a function
// 监听配置项变更
func Init() {
	qpsEventListener := &QPSEventListener{}                       // 限流
	circuitBreakerEventListener := &CircuitBreakerEventListener{} // 熔断
	lbEventListener := &LoadBalancingEventListener{}              // 负载均衡

	RegisterKeys(qpsEventListener, Prefix)
	RegisterKeys(circuitBreakerEventListener, ConsumerFallbackKey, ConsumerFallbackPolicyKey, ConsumerIsolationKey, ConsumerCircuitBreakerKey)
	RegisterKeys(lbEventListener, LoadBalanceKey)
	RegisterKeys(&LagerEventListener{}, LagerLevelKey) // 记录日志的最小级别

}
