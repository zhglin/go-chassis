/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chassis

import (
	"fmt"
	"github.com/go-chassis/go-chassis/v2/core/governance"
	"os"
	"sync"

	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/v2/bootstrap"
	"github.com/go-chassis/go-chassis/v2/configserver"
	"github.com/go-chassis/go-chassis/v2/control"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/handler"
	"github.com/go-chassis/go-chassis/v2/core/loadbalancer"
	"github.com/go-chassis/go-chassis/v2/core/registry"
	"github.com/go-chassis/go-chassis/v2/core/router"
	"github.com/go-chassis/go-chassis/v2/core/server"
	"github.com/go-chassis/go-chassis/v2/core/tracing"

	"github.com/go-chassis/go-chassis/v2/pkg/backends/quota"
	"github.com/go-chassis/go-chassis/v2/pkg/metrics"
	"github.com/go-chassis/go-chassis/v2/pkg/runtime"
	"github.com/go-chassis/openlog"
)

type chassis struct {
	schemas     []*Schema
	mu          sync.Mutex
	Initialized bool // 是否已初始化

	DefaultConsumerChainNames map[string]string // 默认的consumerChain
	DefaultProviderChainNames map[string]string // 默认的providerChain

	sigs                   []os.Signal
	preShutDownFuncs       map[string]func(os.Signal)
	postShutDownFuncs      map[string]func(os.Signal)
	hijackGracefulShutdown func(os.Signal)
}

// Schema struct for to represent schema info
type Schema struct {
	serverName string // 协议名称
	schema     interface{}
	opts       []server.RegisterOption
}

// 初始化chain，
func (c *chassis) initChains(chainType string) error {
	var defaultChainName = "default" // 只获取default的
	var handlerNameMap = map[string]string{defaultChainName: ""}
	switch chainType {
	case common.Provider:
		if providerChainMap := config.GlobalDefinition.ServiceComb.Handler.Chain.Provider; len(providerChainMap) != 0 {
			// 如果没有default的 就设置成默认的
			if _, ok := providerChainMap[defaultChainName]; !ok {
				providerChainMap[defaultChainName] = c.DefaultProviderChainNames[defaultChainName]
			}
			handlerNameMap = providerChainMap
		} else {
			// 没有配置就使用默认的
			handlerNameMap = c.DefaultProviderChainNames
		}
	case common.Consumer:
		if consumerChainMap := config.GlobalDefinition.ServiceComb.Handler.Chain.Consumer; len(consumerChainMap) != 0 {
			if _, ok := consumerChainMap[defaultChainName]; !ok {
				consumerChainMap[defaultChainName] = c.DefaultConsumerChainNames[defaultChainName]
			}
			handlerNameMap = consumerChainMap
		} else {
			handlerNameMap = c.DefaultConsumerChainNames
		}
	}
	openlog.Debug(fmt.Sprintf("init %s's handler map", chainType))
	return handler.CreateChains(chainType, handlerNameMap)
}

// 初始化handler
func (c *chassis) initHandler() error {
	if err := c.initChains(common.Provider); err != nil {
		openlog.Error(fmt.Sprintf("chain int failed: %s", err))
		return err
	}
	if err := c.initChains(common.Consumer); err != nil {
		openlog.Error(fmt.Sprintf("chain int failed: %s", err))
		return err
	}
	openlog.Info("chain init success")
	return nil
}

//Init
// 初始化chassis  加载配置 创建对应的组件实例
func (c *chassis) initialize() error {
	if c.Initialized {
		return nil
	}
	// 读取配置文件
	if err := config.Init(); err != nil {
		openlog.Error("failed to initialize conf: " + err.Error())
		return err
	}
	if err := runtime.Init(); err != nil {
		return err
	}

	// 初始化监控
	if err := metrics.Init(); err != nil {
		return err
	}

	// 初始化chain handler
	err := c.initHandler()
	if err != nil {
		openlog.Error(fmt.Sprintf("handler init failed: %s", err))
		return err
	}

	// 初始化服务器端server
	err = server.Init()
	if err != nil {
		return err
	}
	bootstrap.Bootstrap()
	// 注册中心
	if !archaius.GetBool("servicecomb.registry.disabled", false) {
		err := registry.Enable() // 开启注册中心  服务注册以及更新依赖service的cache
		if err != nil {
			return err
		}
		// 开启balance组件
		strategyName := archaius.GetString("cse.loadbalance.strategy.name", "")
		if err := loadbalancer.Enable(strategyName); err != nil {
			return err
		}
	}

	// 配置中心
	err = configserver.Init()
	if err != nil {
		openlog.Warn("lost config server: " + err.Error())
	}
	// router needs get configs from config-server when init
	// so it must init after bootstrap
	// 加载路由组件并进行初始化
	if err = router.Init(); err != nil {
		return err
	}
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}

	// 服务治理相关的配置管理   负载均衡 限流 熔断 错误注入
	if err := control.Init(opts); err != nil {
		return err
	}

	// 全链路追踪
	if err = tracing.Init(); err != nil {
		return err
	}

	if err := initBackendPlugins(); err != nil {
		return err
	}
	// 监听配置中心 流量打标 限流
	governance.Init()
	c.Initialized = true
	return nil
}

func initBackendPlugins() error {
	if err := quota.Init(quota.Options{
		Plugin:   archaius.GetString("servicecomb.quota.plugin", ""),
		Endpoint: archaius.GetString("servicecomb.quota.endpoint", ""),
	}); err != nil {
		return err
	}
	return nil
}
func (c *chassis) registerSchema(serverName string, structPtr interface{}, opts ...server.RegisterOption) {
	schema := &Schema{
		serverName: serverName,
		schema:     structPtr,
		opts:       opts,
	}
	c.mu.Lock()
	c.schemas = append(c.schemas, schema)
	c.mu.Unlock()
}

func (c *chassis) start() error {
	if !c.Initialized {
		return fmt.Errorf("the chassis do not init. please run chassis.Init() first")
	}

	for _, v := range c.schemas {
		if v == nil {
			continue
		}
		s, err := server.GetServer(v.serverName)
		if err != nil {
			return err
		}
		_, err = s.Register(v.schema, v.opts...)
		if err != nil {
			return err
		}
	}
	err := server.StartServer()
	if err != nil {
		return err
	}
	return nil
}
