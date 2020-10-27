package loadbalancer

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-chassis/openlog"
)

// 支持的所有balancer方式
var strategies = make(map[string]func() Strategy)
var i int // session_stickiness使用

func init() {
	rand.Seed(time.Now().UnixNano())
	rand.Seed(time.Now().Unix())
	i = rand.Int()
}

// InstallStrategy install strategy
// 添加loadBalancer组件
func InstallStrategy(name string, s func() Strategy) {
	strategies[name] = s
	openlog.Debug(fmt.Sprintf("installed strategy plugin: %s.", name))
}

// GetStrategyPlugin get strategy plugin
// 获取指定的 balance strategy
func GetStrategyPlugin(name string) (func() Strategy, error) {
	s, ok := strategies[name]
	if !ok {
		return nil, fmt.Errorf("don't support strategyName [%s]", name)
	}

	return s, nil
}
