package handler

import (
	"fmt"
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/openlog"
)

// ChainMap just concurrent read
// 创建好的chain 全局保存 key = chainType+chainName
var ChainMap = make(map[string]*Chain)

// Chain struct for service and handlers
type Chain struct {
	ServiceType string    // 类型 consumer || provider
	Name        string    // 标识
	Handlers    []Handler // handlers
}

func (c *Chain) Clone() Chain {
	var clone = Chain{
		ServiceType: c.ServiceType,
		Name:        c.Name,
		Handlers:    make([]Handler, len(c.Handlers)),
	}

	for i, h := range c.Handlers {
		clone.Handlers[i] = h
	}
	return clone
}

// AddHandler chain can add a handler
// 添加handler
func (c *Chain) AddHandler(h Handler) {
	c.Handlers = append(c.Handlers, h)
}

// Next is for to handle next handler in the chain
// 执行chain中的handler
func (c *Chain) Next(i *invocation.Invocation, f invocation.ResponseCallBack) {
	index := i.HandlerIndex
	if index >= len(c.Handlers) {
		r := &invocation.Response{
			Err: nil,
		}
		f(r)
		return
	}
	i.HandlerIndex++
	c.Handlers[index].Handle(c, i, f)
}

// ChainOptions chain options
type ChainOptions struct {
	Name string
}

// ChainOption is a function name
type ChainOption func(*ChainOptions)

// WithChainName returns the name of the chain option
func WithChainName(name string) ChainOption {
	return func(c *ChainOptions) {
		c.Name = name
	}
}

// parseHandlers for parsing the handlers
// 过滤不合法的空字符串
func parseHandlers(handlerStr string) []string {
	formatNames := strings.Replace(strings.TrimSpace(handlerStr), " ", "", -1)
	handlerNames := strings.Split(formatNames, ",")
	var s []string
	//delete empty string
	for _, v := range handlerNames {
		if v != "" {
			s = append(s, v)
		}
	}
	return s
}

//CreateChains create the chains based on type and handler map
// 创建指定的chain，handlerNameMap的value是逗号分隔的
func CreateChains(chainType string, handlerNameMap map[string]string) error {
	for chainName := range handlerNameMap {
		handlerNames := parseHandlers(handlerNameMap[chainName]) // 过滤并转换成数组
		c, err := CreateChain(chainType, chainName, handlerNames...)
		if err != nil {
			return fmt.Errorf("err create chain %s.%s:%s %s", chainType, chainName, handlerNames, err.Error())
		}
		// 添加到全局变量
		ChainMap[chainType+chainName] = c

	}
	return nil
}

//CreateChain create consumer or provider's chain,the handlers is different
// 创建chain serviceType = consumer || provider
func CreateChain(serviceType string, chainName string, handlerNames ...string) (*Chain, error) {
	c := &Chain{
		ServiceType: serviceType,
		Name:        chainName,
	}
	openlog.Debug(fmt.Sprintf("add [%d] handlers for chain [%s]", len(handlerNames), chainName))

	// 依次添加handler
	for _, name := range handlerNames {
		err := addHandler(c, name)
		if err != nil {
			return nil, err
		}
	}

	if len(c.Handlers) == 0 {
		openlog.Warn("Chain " + chainName + " is Empty")
		return c, nil
	}
	return c, nil
}

// addHandler add handler
// 依次创建handler并添加到chain中
func addHandler(c *Chain, name string) error {
	handler, err := CreateHandler(name)
	if err != nil {
		return err
	}
	c.AddHandler(handler)
	return nil
}

// GetChain is to get chain
// 获取已创建好的chain
func GetChain(serviceType string, name string) (*Chain, error) {
	if name == "" {
		name = common.DefaultChainName
	}
	origin, ok := ChainMap[serviceType+name]
	if !ok {
		return nil, fmt.Errorf("get chain [%s] failed", serviceType+name)
	}
	return origin, nil
}
