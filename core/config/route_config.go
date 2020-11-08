package config

import "github.com/go-chassis/go-archaius"

//DefaultRouterType set the default router type
const DefaultRouterType = "cse" // 默认使用的router组件

// GetRouterType returns the type of router
func GetRouterType() string {
	return archaius.GetString("servicecomb.router.infra", DefaultRouterType)
}

// GetRouterEndpoints returns the router address
func GetRouterEndpoints() string {
	return archaius.GetString("servicecomb.router.address", "")
}
