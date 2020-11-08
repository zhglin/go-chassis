package handler

import (
	"errors"
	"fmt"

	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/pkg/string"
)

var errViolateBuildIn = errors.New("can not replace build-in handler func")

//ErrDuplicatedHandler means you registered more than 1 handler with same name
var ErrDuplicatedHandler = errors.New("duplicated handler registration")
var buildIn = []string{Loadbalance, Router, TracingConsumer,
	TracingProvider, Transport, FaultInject}

// HandlerFuncMap handler function map
var HandlerFuncMap = make(map[string]func() Handler)

// constant keys for handlers
const (
	//consumer chain
	Transport       = "transport"
	Loadbalance     = "loadbalance"
	TracingConsumer = "tracing-consumer"

	Router             = "router"
	FaultInject        = "fault-inject"
	SkyWalkingConsumer = "skywalking-consumer"

	//provider chain
	RateLimiterProvider = "ratelimiter-provider"
	TracingProvider     = "tracing-provider"
	SkyWalkingProvider  = "skywalking-provider"
)

// init is for to initialize the all handlers at boot time
// 初始化 设置所有handler对应的创建函数
func init() {
	//register build-in handler,don't need to call RegisterHandlerFunc
	HandlerFuncMap[Transport] = newTransportHandler
	HandlerFuncMap[Loadbalance] = newLBHandler                  // 负载军黑
	HandlerFuncMap[TracingProvider] = newTracingProviderHandler // 服务器端追踪
	HandlerFuncMap[TracingConsumer] = newTracingConsumerHandler // 客户端追踪
	HandlerFuncMap[Router] = newRouterHandler                   // 路由
	HandlerFuncMap[FaultInject] = newFaultHandler               // 故障注入
	HandlerFuncMap[TrafficMarker] = newMarkHandler              // 流量打标
}

// Handler interface for handlers
// handler接口
type Handler interface {
	// handle invocation transportation,and tr response
	Handle(*Chain, *invocation.Invocation, invocation.ResponseCallBack)
	Name() string
}

//WriteBackErr write err and callback
func WriteBackErr(err error, status int, cb invocation.ResponseCallBack) {
	r := &invocation.Response{
		Err:    err,
		Status: status,
	}
	cb(r)
}

// RegisterHandler Let developer custom handler
// 动态注册handler
func RegisterHandler(name string, f func() Handler) error {
	if stringutil.StringInSlice(name, buildIn) {
		return errViolateBuildIn
	}
	_, ok := HandlerFuncMap[name]
	if ok {
		return ErrDuplicatedHandler
	}
	HandlerFuncMap[name] = f
	return nil
}

// CreateHandler create a new handler by name your registered
// 创建指定name的handler
func CreateHandler(name string) (Handler, error) {
	f := HandlerFuncMap[name]
	if f == nil {
		return nil, fmt.Errorf("don't have handler [%s]", name)
	}
	return f(), nil
}
