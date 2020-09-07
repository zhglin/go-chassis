package eventlistener

import (
	"fmt"
	"github.com/go-chassis/go-archaius/event"
	"github.com/go-chassis/go-chassis/v2/control/servicecomb"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/openlog"
)

// constants for loadbalancer strategy name, and timeout
const (
	//LoadBalanceKey is variable of type string that matches load balancing events
	LoadBalanceKey = "^cse\\.loadbalance\\."
)

//LoadbalancingEventListener is a struct
type LoadbalancingEventListener struct {
	Key string
}

//Event is a method used to handle a load balancing event
func (e *LoadbalancingEventListener) Event(evt *event.Event) {
	openlog.Debug(fmt.Sprintf("LB event, key: %s, type: %s", evt.Key, evt.EventType))
	if err := config.ReadLBFromArchaius(); err != nil {
		openlog.Error("can not unmarshal new lb config: " + err.Error())
	}
	servicecomb.SaveToLBCache(config.GetLoadBalancing())
}
