// Package router expose API for user to get or set route rule
package router

import (
	"errors"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/marker"
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	wp "github.com/go-chassis/go-chassis/v2/core/router/weightpool"
	"github.com/go-chassis/openlog"
)

//Router return route rule, you can also set custom route rule
// router接口
type Router interface {
	// 初始化
	Init(Options) error
	// 设置所有service的rule
	SetRouteRule(map[string][]*config.RouteRule)
	// 获取指定service的rule
	FetchRouteRuleByServiceName(service string) []*config.RouteRule
	ListRouteRule() map[string][]*config.RouteRule
}

// ErrNoExist means if there is no router implementation
var ErrNoExist = errors.New("router not exists")
var routerServices = make(map[string]func() (Router, error))

// DefaultRouter is current router implementation
var DefaultRouter Router

// InstallRouterPlugin install router plugin
// 注册router组件
func InstallRouterPlugin(name string, f func() (Router, error)) {
	openlog.Info("install route rule plugin: " + name)
	routerServices[name] = f
}

//BuildRouter create a router
// 创建指定name的默认router
func BuildRouter(name string) error {
	f, ok := routerServices[name]
	if !ok {
		return ErrNoExist
	}
	r, err := f()
	if err != nil {
		return err
	}
	DefaultRouter = r
	return nil
}

//Route decide the target service metadata
//it decide based on configuration of route rule
//it will set RouteTag to invocation
// 匹配route  header请求的head
func Route(header map[string]string, si *registry.SourceInfo, inv *invocation.Invocation) error {
	rules := SortRules(inv.MicroServiceName) // 获取配置
	for _, rule := range rules {
		if Match(inv, rule.Match, header, si) { // 每条规则进行匹配
			tag := FitRate(rule.Routes, inv.MicroServiceName)
			// inv里面设置routeTag
			inv.RouteTags = routeTagToTags(tag)
			break
		}
	}
	return nil
}

// FitRate fit rate
// 获取目标服务的weightPool 并 选选择一个routeTag
func FitRate(tags []*config.RouteTag, dest string) *config.RouteTag {
	if tags[0].Weight == 100 {
		return tags[0]
	}

	pool, ok := wp.GetPool().Get(dest)
	if !ok {
		// first request route to tags[0]  首次直接返回第一个
		wp.GetPool().Set(dest, wp.NewPool(tags...))
		return tags[0]
	}
	return pool.PickOne()
}

// match check the route rule
// 是否匹配match
func Match(inv *invocation.Invocation, matchConf config.Match, headers map[string]string, source *registry.SourceInfo) bool {
	//validate template first 匹配已设置的规则 是否参考流量标记
	if refer := matchConf.Refer; refer != "" {
		marker.Mark(inv)
		// 是否能匹配
		return inv.GetMark() == matchConf.Refer
	}
	//matchConf rule is not set 都没有设置 true
	if matchConf.Source == "" && matchConf.HTTPHeaders == nil && matchConf.Headers == nil {
		return true
	}

	return SourceMatch(&matchConf, headers, source)
}

// SourceMatch check the source route
// 是否匹配match tag的匹配要全部都匹配
func SourceMatch(match *config.Match, headers map[string]string, source *registry.SourceInfo) bool {
	//source not match consumer来源
	if match.Source != "" && match.Source != source.Name {
		return false
	}
	//source tags not match invocation.metadata
	if len(match.SourceTags) != 0 {
		for k, v := range match.SourceTags {
			if v != source.Tags[k] {
				return false
			}
		}
	}

	//source headers not match
	if match.Headers != nil {
		for k, v := range match.Headers {
			if !isMatch(headers, k, v) {
				return false
			}
			continue
		}
	}
	if match.HTTPHeaders != nil {
		for k, v := range match.HTTPHeaders {
			if !isMatch(headers, k, v) {
				return false
			}
			continue
		}
	}
	return true
}

// isMatch check the route rule
// header 值 校验
func isMatch(headers map[string]string, k string, v map[string]string) bool {
	header := valueToUpper(v["caseInsensitive"], headers[k])
	for op, exp := range v {
		if op == "caseInsensitive" {
			continue
		}
		if ok, err := marker.Match(op, header, valueToUpper(v["caseInsensitive"], exp)); !ok || err != nil {
			return false
		}
	}
	return true
}

// 是否不区分大小写
func valueToUpper(b, value string) string {
	if b == common.TRUE {
		value = strings.ToUpper(value)
	}

	return value
}

// SortRules sort route rules
// 获取指定service的rule并按优先级排序
func SortRules(name string) []*config.RouteRule {
	if DefaultRouter == nil {
		openlog.Debug("router not available")
	}
	slice := DefaultRouter.FetchRouteRuleByServiceName(name)
	return QuickSort(0, len(slice)-1, slice)
}

// QuickSort for sorting the routes it will follow quicksort technique
// 根据precedence进行排序  双向扫描分区法
func QuickSort(left int, right int, rules []*config.RouteRule) (s []*config.RouteRule) {
	s = rules
	if left >= right {
		return
	}

	i := left
	j := right
	base := s[left]
	var tmp *config.RouteRule
	for i != j {
		for s[j].Precedence <= base.Precedence && i < j {
			j--
		}
		for s[i].Precedence >= base.Precedence && i < j {
			i++
		}
		if i < j {
			tmp = s[i]
			s[i] = s[j]
			s[j] = tmp
		}
	}
	//move base to the current position of i&j
	s[left] = s[i]
	s[i] = base

	QuickSort(left, i-1, s)
	QuickSort(i+1, right, s)

	return
}
