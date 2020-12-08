package servicecomb

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/router"
	"github.com/go-chassis/openlog"
	"sync"
)

var cseRouter *Router

//Router is cse router service
type Router struct {
	// provider对应的路由规则
	routeRule map[string][]*config.RouteRule // service=>[]rule
	lock      sync.RWMutex
}

//SetRouteRule set rules
func (r *Router) SetRouteRule(rr map[string][]*config.RouteRule) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.routeRule = rr
}

//FetchRouteRuleByServiceName get rules for service
// 获取指定service的rule
func (r *Router) FetchRouteRuleByServiceName(service string) []*config.RouteRule {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.routeRule[service]
}

//ListRouteRule get rules for all service
func (r *Router) ListRouteRule() map[string][]*config.RouteRule {
	r.lock.RLock()
	defer r.lock.RUnlock()
	rr := make(map[string][]*config.RouteRule, len(r.routeRule))
	for k, v := range r.routeRule {
		rr[k] = v
	}
	return rr
}

//Init init router config
// 初始化router组件
func (r *Router) Init(o router.Options) error {
	// 配置中心watch
	err := archaius.RegisterListener(&routeRuleEventListener{}, DarkLaunchKey, DarkLaunchKeyV2)
	if err != nil {
		openlog.Error(err.Error())
	}
	// 设置rule配置
	return r.LoadRules()
}

// 创建cse类型的router
func newRouter() (router.Router, error) {
	cseRouter = &Router{
		routeRule: make(map[string][]*config.RouteRule),
		lock:      sync.RWMutex{},
	}
	return cseRouter, nil
}

// LoadRules load all the router config
// 初始化router配置
func (r *Router) LoadRules() error {
	// 加载配置
	configs, err := MergeLocalAndRemoteConfig()
	if err != nil {
		openlog.Error("init route rule failed", openlog.WithTags(openlog.Tags{
			"err": err.Error(),
		}))
	}

	// 校验 设置
	if router.ValidateRule(configs) {
		r.routeRule = configs
		openlog.Info("load route rule", openlog.WithTags(openlog.Tags{
			"rule": r.routeRule,
		}))
	}
	return nil
}

// SetRouteRuleByKey set route rule by key
// 设置service对应的rule
func (r *Router) SetRouteRuleByKey(k string, rr []*config.RouteRule) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.routeRule[k] = rr
	openlog.Info("update route rule success", openlog.WithTags(
		openlog.Tags{
			"service": k,
			"rule":    rr,
		}))
}

// DeleteRouteRuleByKey set route rule by key
func (r *Router) DeleteRouteRuleByKey(k string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.routeRule, k)
	openlog.Info("route rule is removed", openlog.WithTags(
		openlog.Tags{
			"service": k,
		}))
}

// 注册route组件
func init() {
	router.InstallRouterPlugin("cse", newRouter)
}
