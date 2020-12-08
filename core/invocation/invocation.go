package invocation

import (
	"context"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/pkg/runtime"
	"github.com/go-chassis/go-chassis/v2/pkg/util/tags"
)

// constant values for consumer and provider
const (
	Consumer = iota
	Provider
)
const (
	MDMark = "mark"
)

// Response is invocation response struct
type Response struct {
	Status int
	Result interface{}
	Err    error
}

// ResponseCallBack process invocation response
type ResponseCallBack func(*Response)

//Invocation is the basic struct that used in go chassis to make client and transport layer transparent .
//developer should implements a client which is able to transfer invocation to there own request
//a protocol server should transfer request to invocation and then back to request
type Invocation struct {
	HandlerIndex       int                    // 执行的handler序号
	SSLEnable          bool                   // 是否开启ssl
	Endpoint           string                 //service's ip and port, it is decided in load balancing 请求地址
	Protocol           string                 // 协议标识名 rest ||
	Port               string                 //Port is the name of a real service port (端口号 url里解析出来的端口号)
	SourceServiceID    string                 // 当前service的serviceId
	SourceMicroService string                 // 来源的service
	MicroServiceName   string                 //Target micro service name  当前依赖的服务名称 目标服务名
	SchemaID           string                 //correspond struct name
	OperationID        string                 //correspond struct func name  // request url path
	Args               interface{}            // 请求 http request
	URLPathFormat      string                 // url path
	Reply              interface{}            // 响应 http response 创建Invocation时就创建了Reply
	Ctx                context.Context        //ctx can save protocol headers  存储协议headers 请求调用时设置到header中
	Metadata           map[string]interface{} //local scope data  需要额外记录的数据，提供给外部使用 例如trace MDMark router
	RouteTags          utiltags.Tags          //route tags is decided in router handler // 路由添加 balance route [common.BuildinTagVersion]
	Strategy           string                 //load balancing strategy		负载均衡算法
	Filters            []string               // 对依赖服务的instance进行过滤的函数名
}

//GetMark return match rule name that request matches
func (inv *Invocation) GetMark() string {
	m, ok := inv.Metadata[MDMark].(string)
	if ok {
		return m
	}
	return "none"
}

//Mark marks a invocation, it means the invocation matches a match rule
//so that governance rule can be applied to invocation with specific mark
// 设置匹配到的流量标记
func (inv *Invocation) Mark(matchRuleName string) {
	inv.Metadata[MDMark] = matchRuleName
}

// New create invocation, context can not be nil
// if you don't set ContextHeaderKey, then New will init it
// 创建invocation 一个请求对应一个invocation
func New(ctx context.Context) *Invocation {
	inv := &Invocation{
		SourceServiceID: runtime.ServiceID,
		Ctx:             ctx,
	}
	if inv.Ctx == nil {
		inv.Ctx = context.TODO()
	}
	if inv.Ctx.Value(common.ContextHeaderKey{}) == nil {
		inv.Ctx = context.WithValue(inv.Ctx, common.ContextHeaderKey{}, map[string]string{})
	}
	inv.Metadata = make(map[string]interface{}, 1)
	inv.Metadata[MDMark] = "none"
	return inv
}

//SetMetadata local scope params
// 设置metadata
func (inv *Invocation) SetMetadata(key string, value interface{}) {
	if inv.Metadata == nil {
		inv.Metadata = make(map[string]interface{})
	}
	inv.Metadata[key] = value
}

//SetHeader set headers, the client and server plugins should use them in protocol headers
//it is convenience but has lower performance than you use Headers[k]=v,
// when you have a batch of kv to set
func (inv *Invocation) SetHeader(k, v string) {
	m := inv.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	m[k] = v
}

//Headers return a map that protocol plugin should deliver in transport
func (inv *Invocation) Headers() map[string]string {
	return inv.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
}

//Header return header value
func (inv *Invocation) Header(name string) string {
	m := inv.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	return m[name]
}
