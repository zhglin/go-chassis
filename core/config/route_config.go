package config

import "github.com/go-chassis/go-archaius"

//DefaultRouterType set the default router type
const DefaultRouterType = "cse" // 默认使用的router组件

// GetRouterType returns the type of router
// 区分不同的路由组件  配置结构体中没有
func GetRouterType() string {
	return archaius.GetString("servicecomb.router.infra", DefaultRouterType)
}

// GetRouterEndpoints returns the router address
func GetRouterEndpoints() string {
	return archaius.GetString("servicecomb.router.address", "")
}
