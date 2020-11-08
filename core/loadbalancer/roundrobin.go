package loadbalancer

import (
	"math/rand"
	"sync"

	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/registry"
)

// RoundRobinStrategy is strategy
// 轮询
type RoundRobinStrategy struct {
	instances []*registry.MicroServiceInstance
	key       string
}

func newRoundRobinStrategy() Strategy {
	return &RoundRobinStrategy{}
}

//ReceiveData receive data
func (r *RoundRobinStrategy) ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceKey string) {
	r.instances = instances
	r.key = serviceKey
}

//Pick return instance
func (r *RoundRobinStrategy) Pick() (*registry.MicroServiceInstance, error) {
	if len(r.instances) == 0 {
		return nil, ErrNoneAvailableInstance
	}

	i := pick(r.key)
	return r.instances[i%len(r.instances)], nil
}

// 每个invocation都会新创建balance rrIdxMap全局保留所有服务对应的随机值
var rrIdxMap = make(map[string]int)
var mu sync.RWMutex

func pick(key string) int {
	mu.RLock()
	i, ok := rrIdxMap[key]
	if !ok {
		mu.RUnlock()
		mu.Lock()
		i, ok = rrIdxMap[key]
		if !ok {
			i = rand.Int() // 初始化随机值
			rrIdxMap[key] = i
		}
		rrIdxMap[key]++
		mu.Unlock()
		return i
	}

	mu.RUnlock()
	mu.Lock()
	rrIdxMap[key]++ // 每次加1
	mu.Unlock()
	return i
}
