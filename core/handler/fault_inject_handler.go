package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/config/model"
	"github.com/go-chassis/go-chassis/v2/core/fault"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/openlog"
)

// constant for fault handler name
const (
	FaultHandlerName = "fault-inject"
)

// FaultHandler handler
type FaultHandler struct{}

// newFaultHandler fault handle gives the object of FaultHandler
func newFaultHandler() Handler {
	return &FaultHandler{}
}

// Name function returns fault-inject string
// 错误注入
func (rl *FaultHandler) Name() string {
	return "fault-inject"
}

// Handle is to handle the API
func (rl *FaultHandler) Handle(chain *Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {
	// 配置
	faultConfig := GetFaultConfig(inv.Protocol, inv.MicroServiceName, inv.SchemaID, inv.OperationID)

	// 协议对应的函数
	faultInject, ok := fault.Injectors[inv.Protocol]
	r := &invocation.Response{}
	if !ok {
		msg := "fault injection doesn't support for protocol " + inv.Protocol
		openlog.Error(msg)
		r.Err = errors.New(msg)
		cb(r)
		return
	}

	faultValue := faultConfig
	err := faultInject(faultValue, inv)
	if err != nil {
		if strings.Contains(err.Error(), "injecting abort") { // 执行的是终止
			switch inv.Reply.(type) {
			case *http.Response:
				resp := inv.Reply.(*http.Response)
				resp.StatusCode = faultConfig.Abort.HTTPStatus
			}
			r.Status = faultConfig.Abort.HTTPStatus
		} else { // 延迟
			switch inv.Reply.(type) {
			case *http.Response:
				resp := inv.Reply.(*http.Response)
				resp.StatusCode = http.StatusBadRequest
			}
			r.Status = http.StatusBadRequest
		}

		r.Err = fault.Fault{Message: err.Error()}
		cb(r)
		return
	}

	chain.Next(inv, cb)
}

// GetFaultConfig get faultconfig
func GetFaultConfig(protocol, microServiceName, schemaID, operationID string) model.Fault {

	faultStruct := model.Fault{}
	faultStruct.Abort.Percent = config.GetAbortPercent(protocol, microServiceName, schemaID, operationID)
	faultStruct.Abort.HTTPStatus = config.GetAbortStatus(protocol, microServiceName, schemaID, operationID)
	faultStruct.Delay.Percent = config.GetDelayPercent(protocol, microServiceName, schemaID, operationID)
	faultStruct.Delay.FixedDelay = config.GetFixedDelay(protocol, microServiceName, schemaID, operationID)

	return faultStruct
}
