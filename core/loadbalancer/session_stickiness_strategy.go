package loadbalancer

import (
	"sync"

	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/go-chassis/v2/session"
)

var (

	// successiveFailureCount success and failure count
	successiveFailureCount      map[string]int
	successiveFailureCountMutex sync.RWMutex
)

func init() {
	successiveFailureCount = make(map[string]int)
}

//DeleteSuccessiveFailureCount deleting cookie from failure count map
func DeleteSuccessiveFailureCount(cookieValue string) {
	successiveFailureCountMutex.Lock()
	//	successiveFailureCount[ep] = 0
	delete(successiveFailureCount, cookieValue)
	successiveFailureCountMutex.Unlock()
}

//ResetSuccessiveFailureMap make map again
func ResetSuccessiveFailureMap() {
	successiveFailureCountMutex.Lock()
	successiveFailureCount = make(map[string]int)
	successiveFailureCountMutex.Unlock()
}

//IncreaseSuccessiveFailureCount increase failure count
// 计数
func IncreaseSuccessiveFailureCount(cookieValue string) {
	successiveFailureCountMutex.Lock()
	c, ok := successiveFailureCount[cookieValue]
	if ok {
		successiveFailureCount[cookieValue] = c + 1
		successiveFailureCountMutex.Unlock()
		return
	}
	successiveFailureCount[cookieValue] = 1
	successiveFailureCountMutex.Unlock()
}

//GetSuccessiveFailureCount get failure count
// 获取指定sessionId请求的失败次数
func GetSuccessiveFailureCount(cookieValue string) int {
	successiveFailureCountMutex.RLock()
	defer successiveFailureCountMutex.RUnlock()
	return successiveFailureCount[cookieValue]
}

//SessionStickinessStrategy is strategy
// 会话绑定
type SessionStickinessStrategy struct {
	instances []*registry.MicroServiceInstance
	mtx       sync.Mutex
	sessionID string
}

func newSessionStickinessStrategy() Strategy {
	return &SessionStickinessStrategy{}
}

// ReceiveData receive data
func (r *SessionStickinessStrategy) ReceiveData(inv *invocation.Invocation, instances []*registry.MicroServiceInstance, serviceName string) {
	r.instances = instances
	r.sessionID = session.GetSessionID(getNamespace(inv)) // 当前调用的sessionId
}

// 获取当前invocation的session命名空间  namespace只是区分不同invocation
func getNamespace(i *invocation.Invocation) string {
	if metadata, ok := i.Metadata[common.SessionNameSpaceKey]; ok {
		if v, ok := metadata.(string); ok {
			return v
		}
	}
	return common.SessionNameSpaceDefaultValue
}

// Pick return instance
// 选择实例
func (r *SessionStickinessStrategy) Pick() (*registry.MicroServiceInstance, error) {
	instanceAddr, ok := session.Get(r.sessionID) // 根据sessionId获取address
	if ok {
		// 没有可用节点
		if len(r.instances) == 0 {
			return nil, ErrNoneAvailableInstance
		}

		// sessionId对应的address是否在instance实例中 这里的比较貌似有问题 todo
		for _, instance := range r.instances {
			if instanceAddr == instance.EndpointsMap[instance.DefaultProtocol] {
				return instance, nil
			}
		}
		// if micro service instance goes down then related entry in endpoint map will be deleted,
		//so instead of sending nil, a new instance can be selected using round robin
		// 如果记录的address已经不存在 就轮询选个
		return r.pick()
	}
	return r.pick()

}

// 轮询选一个
func (r *SessionStickinessStrategy) pick() (*registry.MicroServiceInstance, error) {
	if len(r.instances) == 0 {
		return nil, ErrNoneAvailableInstance
	}

	r.mtx.Lock()
	instance := r.instances[i%len(r.instances)]
	i++
	r.mtx.Unlock()

	return instance, nil
}
