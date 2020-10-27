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
	HandlerIndex       int
	SSLEnable          bool
	Endpoint           string //service's ip and port, it is decided in load balancing
	Protocol           string
	Port               string //Port is the name of a real service port
	SourceServiceID    string // 当前service的serviceId
	SourceMicroService string
	MicroServiceName   string //Target micro service name  当前依赖的服务名称
	SchemaID           string //correspond struct name
	OperationID        string //correspond struct func name  url path
	Args               interface{} // 请求 http request
	URLPathFormat      string // url path
	Reply              interface{} // 响应 http response
	Ctx                context.Context        //ctx can save protocol headers  存储协议headers
	Metadata           map[string]interface{} //local scope data  需要额外记录的数据，提供给外部使用 例如trace
	RouteTags          utiltags.Tags          //route tags is decided in router handler
	Strategy           string                 //load balancing strategy
	Filters            []string					// 对instance进行过滤
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
