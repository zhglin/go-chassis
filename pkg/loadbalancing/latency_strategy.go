package loadbalancing

import (
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/loadbalancer"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/openlog"
)

var i int
var weightedRespMutex sync.Mutex

func init() {
	// 根据响应时间，选择平均响应时间最短的
	loadbalancer.InstallStrategy(loadbalancer.StrategyLatency, newWeightedResponseStrategy)
}

// ByDuration is for calculating the duration
type ByDuration []*loadbalancer.ProtocolStats

func (a ByDuration) Len() int           { return len(a) }
func (a ByDuration) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDuration) Less(i, j int) bool { return a[i].AvgLatency < a[j].AvgLatency }

// SortLatency sort instance based on  the average latencies
// 按平均响应时间进行排序
func SortLatency() {
	loadbalancer.LatencyMapRWMutex.RLock()
	for _, v := range loadbalancer.ProtocolStatsMap {
		sort.Sort(ByDuration(v))
	}
	loadbalancer.LatencyMapRWMutex.RUnlock()

}

// CalculateAvgLatency Calculating the average latency for each instance using the statistics collected,
// key is addr/service/protocol
// 计算每个address的平均耗时
func CalculateAvgLatency() {
	loadbalancer.LatencyMapRWMutex.RLock()
	for _, v := range loadbalancer.ProtocolStatsMap {
		for _, stats := range v {
			stats.CalculateAverageLatency()
		}
	}
	loadbalancer.LatencyMapRWMutex.RUnlock()
}

// WeightedResponseStrategy is a strategy plugin
type WeightedResponseStrategy struct {
	instances   []*registry.MicroServiceInstance
	serviceName string
	protocol    string
	tags        string
}

func init() {
	ticker := time.NewTicker(30 * time.Second)
	//run routine to prepare data
	go func() {
		for range ticker.C {
			if config.GetLoadBalancing() != nil {
				useLatencyAware := false
				for _, v := range config.GetLoadBalancing().AnyService {
					if v.Strategy["name"] == loadbalancer.StrategyLatency {
						useLatencyAware = true
						break
					}
				}
				if config.GetLoadBalancing().Strategy["name"] == loadbalancer.StrategyLatency {
					useLatencyAware = true
				}
				// 是否存在使用latencyStrategy方式的service
				if useLatencyAware {
					CalculateAvgLatency() // 计算平均耗时
					SortLatency()         // 按平均耗时进行排序
					openlog.Info("Preparing data for Weighted Response Strategy")
				}
			}

		}
	}()
}

func newWeightedResponseStrategy() loadbalancer.Strategy {
	return &WeightedResponseStrategy{}
}

// ReceiveData receive data
func (r *WeightedResponseStrategy) ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceKey string) {
	r.instances = instances
	keys := strings.SplitN(serviceKey, "|", 2)
	switch len(keys) {
	case 1:
		r.serviceName = keys[0]
	case 2:
		r.serviceName = keys[0]
		r.tags = keys[1]

	}
	r.protocol = inv.Protocol
}

// Pick return instance
func (r *WeightedResponseStrategy) Pick() (*registry.MicroServiceInstance, error) {
	// 70%的流量使用响应时间的加权方式
	if rand.Intn(100) < 70 {
		var instanceAddr string
		// 获取响应时间最短的address
		loadbalancer.LatencyMapRWMutex.RLock()
		if len(loadbalancer.ProtocolStatsMap[loadbalancer.BuildKey(r.serviceName, r.tags, r.protocol)]) != 0 {
			instanceAddr = loadbalancer.ProtocolStatsMap[loadbalancer.BuildKey(r.serviceName, r.tags, r.protocol)][0].Addr
		}
		loadbalancer.LatencyMapRWMutex.RUnlock()
		// 判断此address对应的instance是否已存在
		for _, instance := range r.instances {
			if len(instanceAddr) != 0 && strings.Contains(instance.EndpointsMap[r.protocol].GenEndpoint(), instanceAddr) {
				return instance, nil
			}
		}
	}

	// 未记录到响应时间之前或者对应的address实例已下线 轮询选择
	//if no instances are selected round robin will be done
	weightedRespMutex.Lock()
	node := r.instances[i%len(r.instances)]
	i++
	weightedRespMutex.Unlock()
	return node, nil

}
