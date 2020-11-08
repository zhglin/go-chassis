package fault

import (
	"github.com/go-chassis/go-chassis/v2/core/config/model"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
)

// InjectFault inject fault
// 故障注入 接口
type InjectFault func(model.Fault, *invocation.Invocation) error

// Injectors fault injectors
// 不同协议对应不同的接口
var Injectors = make(map[string]InjectFault)

//Fault fault injection error
type Fault struct {
	Message string
}

func (e Fault) Error() string {
	return e.Message
}

// InstallFaultInjectionPlugin install fault injection plugin
// 注册
func InstallFaultInjectionPlugin(name string, f InjectFault) {
	Injectors[name] = f
}

// 初始化函数
func init() {
	InstallFaultInjectionPlugin("rest", faultInject)
	InstallFaultInjectionPlugin("dubbo", faultInject)
}

func faultInject(rule model.Fault, inv *invocation.Invocation) error {
	return ValidateAndApplyFault(&rule, inv)
}
