package registry

import (
	"errors"
	"fmt"
	"time"

	chassisClient "github.com/go-chassis/go-chassis/v2/core/client"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/openlog"
)

const (
	timeoutToPending = 1 * time.Second
	timeoutToPackage = 100 * time.Millisecond
	chanCapacity     = 1000
)

var defaultHealthChecker = &HealthChecker{}

func init() {
	defaultHealthChecker.Run() // 初始化并执行checker
}

// WrapInstance is the struct defines an instance object with appID/serviceName/version
type WrapInstance struct {
	AppID       string
	ServiceName string
	Version     string
	Instance    *MicroServiceInstance
}

// String is the method returns the string type current instance's key value
func (i *WrapInstance) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", i.ServiceName, i.Version, i.AppID, i.Instance.InstanceID)
}

// ServiceKey is the method returns the string type current instance's service key value
func (i *WrapInstance) ServiceKey() string {
	return fmt.Sprintf("%s:%s:%s", i.ServiceName, i.Version, i.AppID)
}

// HealthChecker is the struct judges the instance health in the removing simpleCache
// 检查被删除的instance
type HealthChecker struct {
	pendingCh chan *WrapInstance // 需要检查的instance
	delCh     chan map[string]*WrapInstance
}

// Run is the method initializes and starts the health check process
// 执行检查
func (hc *HealthChecker) Run() {
	hc.pendingCh = make(chan *WrapInstance, chanCapacity)
	hc.delCh = make(chan map[string]*WrapInstance, chanCapacity)
	go hc.wait()
}

// Add is the method adds a key of the instance simpleCache into pending chan
// 添加需要进行检查的instance
func (hc *HealthChecker) Add(i *WrapInstance) error {
	select {
	case hc.pendingCh <- i:
	case <-time.After(timeoutToPending): // 超时丢弃
		return errors.New("health checker is too busy")
	}
	return nil
}

// 收集事件
func (hc *HealthChecker) wait() {
	pack := make(map[string]*WrapInstance)
	for {
		select {
		case i, ok := <-hc.pendingCh:
			if !ok {
				// chan closed
				return
			}
			pack[i.String()] = i
		case <-time.After(timeoutToPackage): // 收集timeoutToPackage时间的事件 写入delCH
			if len(pack) > 0 {
				hc.delCh <- pack
				pack = make(map[string]*WrapInstance)
			}
		}
	}
}

// HealthCheck is the function adds the instance to HealthChecker
func HealthCheck(service, version, appID string, instance *MicroServiceInstance) error {
	if !config.GetServiceDiscoveryHealthCheck() {
		return fmt.Errorf("health check is disabled")
	}

	return defaultHealthChecker.Add(&WrapInstance{
		ServiceName: service,
		Version:     version,
		AppID:       appID,
		Instance:    instance,
	})
}

// RefreshCache is the function to filter changes between new pulling instances and simpleCache
// 刷新cache 清理下线的instance
func RefreshCache(service string, ups []*MicroServiceInstance, downs map[string]struct{}) {
	c, ok := MicroserviceInstanceIndex.Get(service, nil)
	// 不存在 直接设置
	if !ok || c == nil {
		// if full new instances or at less one instance, then refresh simpleCache immediately
		MicroserviceInstanceIndex.Set(service, ups)
		openlog.Debug(fmt.Sprintf("Cached [%d] Instances of service [%s]", len(ups), service))
		return
	}

	var (
		saves   []*MicroServiceInstance
		lefts   []*MicroServiceInstance
		exps    = c
		mapUps  = make(map[string]*MicroServiceInstance, len(ups))  // 最新的up状态的instance
		mapExps = make(map[string]*MicroServiceInstance, len(exps)) // 现有的instance
	)

	for _, ins := range ups {
		mapUps[ins.InstanceID] = ins
	}
	for _, instance := range exps {
		mapExps[instance.InstanceID] = instance
	}

	for _, exp := range mapExps {
		// case: keep still alive instances  现有的节点依旧是开启状态
		if _, ok := mapUps[exp.InstanceID]; ok {
			lefts = append(lefts, exp)
			openlog.Debug(fmt.Sprintf("cache instance: %v", exp))
			continue
		} else {
			// 未开启的关闭链接
			for p, ep := range exp.EndpointsMap {
				if err := chassisClient.Close(p, service, ep.GenEndpoint()); err != nil {
					if err != chassisClient.ErrClientNotExist {
						openlog.Error(fmt.Sprintf("can not close [%s] client for service [%s],intance [%s,%s,%s]: %s",
							p, service, exp.InstanceID, ep, exp.HostName, err))
					}
				} else {
					openlog.Debug(fmt.Sprintf("closed [%s] client for service [%s],intance [%s,%s,%s]",
						p, service, exp.InstanceID, ep, exp.HostName))
				}
			}
		}
		// case: remove instances with the non-up status  在downs中 跳过
		if _, ok := downs[exp.InstanceID]; ok {
			continue
		}
		// case: keep instances returned HC ok  即不在up也不在down instance被删除了
		if err := HealthCheck(service, exp.version(), exp.appID(), exp); err == nil {
			lefts = append(lefts, exp)
		}
	}

	for _, up := range ups {
		if _, ok := mapExps[up.InstanceID]; ok {
			continue
		}
		// case: add new come in instances  新加的instance
		saves = append(saves, up)
	}

	// 不存在up的instance
	lefts = append(lefts, saves...)
	if len(lefts) == 0 {
		//todo remove this when the simpleCache struct can delete the key if the input is an empty slice
		MicroserviceInstanceIndex.Delete(service)
		openlog.Info(fmt.Sprintf("Delete the service [%s] in the cache", service))
		return
	}

	// 重置instance
	MicroserviceInstanceIndex.Set(service, lefts)
	openlog.Debug(fmt.Sprintf("Cached [%d] Instances of service [%s]", len(lefts), service))
}
