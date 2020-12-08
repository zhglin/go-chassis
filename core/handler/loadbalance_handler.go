package handler

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/go-chassis/v2/resilience/retry"
	"io/ioutil"
	"net/http"

	//"github.com/cenkalti/backoff"
	"github.com/go-chassis/go-chassis/v2/control"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/loadbalancer"
	"github.com/go-chassis/go-chassis/v2/core/status"
	"github.com/go-chassis/go-chassis/v2/pkg/util"
	"github.com/go-chassis/go-chassis/v2/resilience/retry/backof"
	"github.com/go-chassis/openlog"
)

// LBHandler loadbalancer handler struct
type LBHandler struct{}

// 获取服务的节点
func (lb *LBHandler) getEndpoint(i *invocation.Invocation, lbConfig control.LoadBalancingConfig) (*registry.Endpoint, error) {
	var strategyFun func() loadbalancer.Strategy
	var err error
	// 设置strategyFun 如果invocation中未设置负载均衡算法就使用lb的配置
	if i.Strategy == "" {
		i.Strategy = lbConfig.Strategy
		strategyFun, err = loadbalancer.GetStrategyPlugin(i.Strategy)
		if err != nil {
			openlog.Error(fmt.Sprintf("lb error [%s] because of [%s]", loadbalancer.LBError{
				Message: "Get strategy [" + i.Strategy + "] failed."}.Error(), err.Error()))
		}
	} else {
		strategyFun, err = loadbalancer.GetStrategyPlugin(i.Strategy)
		if err != nil {
			openlog.Error(fmt.Sprintf("lb error [%s] because of [%s]", loadbalancer.LBError{
				Message: "Get strategy [" + i.Strategy + "] failed."}.Error(), err.Error()))
		}
	}

	// 设置Filters函数
	if len(i.Filters) == 0 {
		i.Filters = lbConfig.Filters
	}

	// 获取balance实例 每次请求都创建新的实例
	s, err := loadbalancer.BuildStrategy(i, strategyFun())
	if err != nil {
		return nil, err
	}

	// 选择一个instance
	ins, err := s.Pick()
	if err != nil {
		lbErr := loadbalancer.LBError{Message: err.Error()}
		return nil, lbErr
	}

	// 设置默认protocol
	if i.Protocol == "" {
		for k := range ins.EndpointsMap {
			i.Protocol = k
			break
		}
	}

	// 判断ins是否支持protocol
	// ins.EndpointsMap是已protocol为key 这里的port有问题
	protocolServer := util.GenProtoEndPoint(i.Protocol, i.Port)
	ep, ok := ins.EndpointsMap[protocolServer]
	if !ok {
		errStr := fmt.Sprintf(
			"No available instance for protocol server [%s] , microservice: %s has %v",
			protocolServer, i.MicroServiceName, ins.EndpointsMap)
		lbErr := loadbalancer.LBError{Message: errStr}
		openlog.Error(lbErr.Error())
		return nil, lbErr
	}
	return ep, nil
}

// Handle to handle the load balancing
func (lb *LBHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	// 获取balance配置
	lbConfig := control.DefaultPanel.GetLoadBalancing(*i)
	if !lbConfig.RetryEnabled {
		lb.handleWithNoRetry(chain, i, lbConfig, cb)
	} else {
		lb.handleWithRetry(chain, i, lbConfig, cb)
	}
}

// 没有重试机制的handle
func (lb *LBHandler) handleWithNoRetry(chain *Chain, i *invocation.Invocation, lbConfig control.LoadBalancingConfig, cb invocation.ResponseCallBack) {
	ep, err := lb.getEndpoint(i, lbConfig)
	if err != nil {
		WriteBackErr(err, status.Status(i.Protocol, status.ServiceUnavailable), cb)
		return
	}

	i.Endpoint = ep.Address // 设置请求地址
	i.SSLEnable = ep.IsSSLEnable()
	chain.Next(i, cb)
}

// 有重试机制的handle
func (lb *LBHandler) handleWithRetry(chain *Chain, i *invocation.Invocation, lbConfig control.LoadBalancingConfig, cb invocation.ResponseCallBack) {
	retryOnSame := lbConfig.RetryOnSame
	retryOnNext := lbConfig.RetryOnNext
	handlerIndex := i.HandlerIndex // 保留下handlerIndex
	var invResp *invocation.Response
	var reqBytes []byte // 额外保存下 防止后面的handler修改掉
	if req, ok := i.Args.(*http.Request); ok {
		if req != nil {
			if req.Body != nil {
				reqBytes, _ = ioutil.ReadAll(req.Body)
			}
		}
	}
	// get retry func
	lbBackoff := retry.GetBackOff(lbConfig.BackOffKind, lbConfig.BackOffMin, lbConfig.BackOffMax)
	callTimes := 0

	ep, err := lb.getEndpoint(i, lbConfig)
	if err != nil {
		// if get endpoint failed, no need to retry
		WriteBackErr(err, status.Status(i.Protocol, status.ServiceUnavailable), cb)
		return
	}
	operation := func() error {
		i.Endpoint = ep.Address
		i.SSLEnable = ep.IsSSLEnable()
		callTimes++
		var respErr error
		i.HandlerIndex = handlerIndex

		if _, ok := i.Args.(*http.Request); ok {
			i.Args.(*http.Request).Body = ioutil.NopCloser(bytes.NewBuffer(reqBytes))
		}

		chain.Next(i, func(r *invocation.Response) {
			if r != nil {
				invResp = r
				respErr = invResp.Err
				return
			}
		})

		// 达到相同的节点调用次数限制就更换节点
		if callTimes >= retryOnSame+1 {
			// 是否需要更换节点
			if retryOnNext <= 0 {
				return backoff.Permanent(errors.New("retry times expires"))
			}
			ep, err = lb.getEndpoint(i, lbConfig)
			if err != nil {
				// if get endpoint failed, no need to retry
				return backoff.Permanent(err)
			}
			callTimes = 0
			retryOnNext--
		}
		return respErr
	}
	if err := backoff.Retry(operation, lbBackoff); err != nil {
		openlog.Error(fmt.Sprintf("stop retry , error : %v", err))
	}

	if invResp == nil {
		invResp = &invocation.Response{}
	}
	cb(invResp)
}

// Name returns loadbalancer string
func (lb *LBHandler) Name() string {
	return "loadbalancer"
}

func newLBHandler() Handler {
	return &LBHandler{}
}
