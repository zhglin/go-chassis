// Package loadbalancer is client side load balancer
package loadbalancer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/go-chassis/v2/pkg/util/tags"
	"github.com/go-chassis/openlog"
)

// constant string for zoneaware
const (
	ZoneAware = "zoneaware"
)

//StrategyLatency is name
const StrategyLatency = "WeightedResponse"

// constant strings for load balance variables
const (
	StrategyRoundRobin        = "RoundRobin"
	StrategyRandom            = "Random"
	StrategySessionStickiness = "SessionStickiness"

	OperatorEqual   = "="
	OperatorGreater = ">"
	OperatorSmaller = "<"
	OperatorPattern = "Pattern"
)

var (
	// ErrNoneAvailableInstance is to represent load balance error
	ErrNoneAvailableInstance = LBError{Message: "None available instance"}
)

// LBError load balance error
type LBError struct {
	Message string
}

// Error for to return load balance error message
func (e LBError) Error() string {
	return "lb: " + e.Message
}

// BuildStrategy query instance list and give it to Strategy then return Strategy
// 创建并填充balance
func BuildStrategy(i *invocation.Invocation,
	s Strategy) (Strategy, error) {

	// strategy不存在设置默认balance
	if s == nil {
		s = &RoundRobinStrategy{}
	}

	var isFilterExist = true
	for _, filter := range i.Filters {
		if filter == "" {
			isFilterExist = false
		}
	}

	// 获取instances
	instances, err := registry.DefaultServiceDiscoveryService.FindMicroServiceInstances(i.SourceServiceID, i.MicroServiceName, i.RouteTags)
	if err != nil {
		lbErr := LBError{err.Error()}
		openlog.Error(fmt.Sprintf("Lb err: %s", err))
		return nil, lbErr
	}

	// 过滤instance
	if isFilterExist {
		filterFuncs := make([]Filter, 0)
		//append filters in config
		for _, fName := range i.Filters {
			f := Filters[fName]
			if f != nil {
				filterFuncs = append(filterFuncs, f)
				continue
			}
		}
		for _, filter := range filterFuncs {
			instances = filter(instances, nil)
		}
	}

	if len(instances) == 0 {
		lbErr := LBError{fmt.Sprintf("No available instance, key: %s(%v)", i.MicroServiceName, i.RouteTags)}
		openlog.Error(lbErr.Error())
		return nil, lbErr
	}

	// 填充balance数据
	serviceKey := strings.Join([]string{i.MicroServiceName, i.RouteTags.String()}, "|")
	s.ReceiveData(i, instances, serviceKey)
	return s, nil
}

// Strategy is load balancer algorithm , call Pick to return one instance
// loadBalancer接口
type Strategy interface {
	// 填充数据
	ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceKey string)
	// 返回节点
	Pick() (*registry.MicroServiceInstance, error)
}

//Criteria is rule for filter
type Criteria struct {
	Key      string
	Operator string
	Value    string
}

// Filter receive instances and criteria, it will filter instances based on criteria you defined,criteria is optional, you can give nil for it
// 统一的过滤函数
type Filter func(instances []*registry.MicroServiceInstance, criteria []*Criteria) []*registry.MicroServiceInstance

// Enable function is for to enable load balance strategy
// 添加支持的balancer
func Enable(strategyName string) error {
	openlog.Info("Enable LoadBalancing")
	InstallStrategy(StrategyRandom, newRandomStrategy) // 随机
	InstallStrategy(StrategyRoundRobin, newRoundRobinStrategy) // 轮询
	InstallStrategy(StrategySessionStickiness, newSessionStickinessStrategy) // 会话

	if strategyName == "" {
		openlog.Info("Empty strategy configuration, use RoundRobin as default")
		return nil
	}
	openlog.Info("Strategy is " + strategyName)

	return nil
}

// Filters is a map of string and array of *registry.MicroServiceInstance
// instances的过滤函数
var Filters = make(map[string]Filter)

// InstallFilter install filter
// 注册Filter函数
func InstallFilter(name string, f Filter) {
	Filters[name] = f
}

// variables for latency map, rest and highway requests count
var (
	//ProtocolStatsMap saves all stats for all service's protocol, one protocol has a lot of instances
	ProtocolStatsMap = make(map[string][]*ProtocolStats)
	//maintain different locks since multiple goroutine access the map
	LatencyMapRWMutex sync.RWMutex
)

//BuildKey return key of stats map
func BuildKey(microServiceName, tags, protocol string) string {
	//TODO add more data
	return strings.Join([]string{microServiceName, tags, protocol}, "/")
}

// SetLatency for a instance, it only save latest 10 stats for instance's protocol
func SetLatency(latency time.Duration, addr, microServiceName string, tags utiltags.Tags, protocol string) {
	key := BuildKey(microServiceName, tags.String(), protocol)

	LatencyMapRWMutex.RLock()
	stats, ok := ProtocolStatsMap[key]
	LatencyMapRWMutex.RUnlock()
	if !ok {
		stats = make([]*ProtocolStats, 0)
	}
	exist := false
	for _, v := range stats {
		if v.Addr == addr {
			v.SaveLatency(latency)
			exist = true
		}
	}
	if !exist {
		ps := &ProtocolStats{
			Addr: addr,
		}

		ps.SaveLatency(latency)
		stats = append(stats, ps)
	}
	LatencyMapRWMutex.Lock()
	ProtocolStatsMap[key] = stats
	LatencyMapRWMutex.Unlock()
}
