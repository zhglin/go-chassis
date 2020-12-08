package router

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/config"
	chassisTLS "github.com/go-chassis/go-chassis/v2/core/tls"
	"github.com/go-chassis/go-chassis/v2/pkg/util/iputil"
	"github.com/go-chassis/go-chassis/v2/pkg/util/tags"
	"github.com/go-chassis/openlog"
)

// RouterTLS defines tls prefix
const RouterTLS = "router"

//Init initialize router config in local file
//then is create the router component
// 初始化router配置
func Init() error {
	// 创建默认的router组件
	err := BuildRouter(config.GetRouterType())
	if err != nil {
		openlog.Error("can not init router [" + config.GetRouterType() + "]: " + err.Error())
		return err
	}
	op, err := getSpecifiedOptions()
	if err != nil {
		return fmt.Errorf("router options error: %v", err)
	}
	// 初始化默认的router组件 并加载配置
	err = DefaultRouter.Init(op)
	if err != nil {
		openlog.Error(err.Error())
		return err
	}
	openlog.Info("router init success")
	return nil
}

// ValidateRule validate the route rules of each service
// 处理并校验 rules 有一个异常所有规则不生效
func ValidateRule(rules map[string][]*config.RouteRule) bool {
	for name, rule := range rules {
		for _, route := range rule {
			allWeight := 0
			for _, routeTag := range route.Routes {
				// 设置label
				routeTag.Label = utiltags.LabelOfTags(routeTag.Tags)
				allWeight += routeTag.Weight
			}
			// 校验权重
			if allWeight > 100 {
				openlog.Warn("route rule is invalid: total weight is over 100%", openlog.WithTags(
					openlog.Tags{
						"service": name,
					}))
				return false
			}
		}

	}
	return true
}

// Options defines how to init router and its fetcher
type Options struct {
	Endpoints []string
	EnableSSL bool
	TLSConfig *tls.Config
	Version   string

	//TODO: need timeout for client
	// TimeOut time.Duration
}

func getSpecifiedOptions() (opts Options, err error) {
	hosts, scheme, err := iputil.URIs2Hosts(strings.Split(config.GetRouterEndpoints(), ","))
	if err != nil {
		return
	}
	opts.Endpoints = hosts
	// TODO: envoy api v1 or v2
	// opts.Version = config.GetRouterAPIVersion()
	opts.TLSConfig, err = chassisTLS.GetTLSConfig(scheme, RouterTLS)
	if err != nil {
		return
	}
	if opts.TLSConfig != nil {
		opts.EnableSSL = true
	}
	return
}

// routeTagToTags returns tags from a route tag
// 转换tag
func routeTagToTags(t *config.RouteTag) utiltags.Tags {
	tag := utiltags.Tags{}
	if t != nil {
		tag.KV = make(map[string]string, len(t.Tags))
		for k, v := range t.Tags {
			tag.KV[k] = v
		}
		tag.Label = t.Label
		return tag
	}
	return tag
}
