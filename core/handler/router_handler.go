package handler

import (
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/go-chassis/v2/core/router"
	"github.com/go-chassis/go-chassis/v2/core/status"
	"github.com/go-chassis/go-chassis/v2/pkg/runtime"
)

// RouterHandler router handler
// 路由
type RouterHandler struct{}

// Handle is to handle the router related things
func (ph *RouterHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	// 已设置 跳过 routeTags可以手动指定
	if i.RouteTags.KV != nil {
		chain.Next(i, cb)
		return
	}

	tags := map[string]string{}
	for k, v := range i.Metadata {
		tags[k] = v.(string)
	}
	tags[common.BuildinTagApp] = runtime.App

	h := make(map[string]string)
	if i.Ctx != nil {
		at, ok := i.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
		if ok {
			h = at
		}
	}

	// 匹配路由规则并设置对应的路由标记
	err := router.Route(h, &registry.SourceInfo{Name: i.SourceMicroService, Tags: tags}, i)
	if err != nil {
		WriteBackErr(err, status.Status(i.Protocol, status.ServiceUnavailable), cb)
	}

	//call next chain
	chain.Next(i, cb)
}

func newRouterHandler() Handler {
	return &RouterHandler{}
}

// Name returns the router string
func (ph *RouterHandler) Name() string {
	return "router"
}
