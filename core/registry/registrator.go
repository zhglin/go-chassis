package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/openlog"
)

// constant values for registry parameters
const (
	DefaultRegistratorPlugin       = "servicecenter"
	DefaultServiceDiscoveryPlugin  = "servicecenter"
	DefaultContractDiscoveryPlugin = "servicecenter"
	Name                           = "registry"
	SDTag                          = "serviceDiscovery"
	CDTag                          = "contractDiscovery"
	RTag                           = "registrator"
	Auto                           = "auto"
	Manual                         = "manual"
)

// IsEnabled check enable
var IsEnabled bool  // 是否已开启
var mu sync.Mutex

// DefaultRegistrator is the client of registry, you can call the method of it to interact with microservice registry
var DefaultRegistrator Registrator

// DefaultAddr default address of service center
var DefaultAddr = "http://127.0.0.1:30100"

// registryFunc registry function
var registryFunc = make(map[string]func(opts Options) Registrator)

// HBService variable of heartbeat service
var HBService = &HeartbeatService{}

// Registrator is the interface for developer to update information in service registry
type Registrator interface {
	//Close destroy connection between the registry client and server
	Close() error
	//RegisterService register a microservice to registry, if it is duplicated in registry, it returns error
	RegisterService(microService *MicroService) (string, error)
	//RegisterServiceInstance register a microservice instance to registry
	RegisterServiceInstance(sid string, instance *MicroServiceInstance) (string, error)
	RegisterServiceAndInstance(microService *MicroService, instance *MicroServiceInstance) (string, string, error)
	Heartbeat(microServiceID, microServiceInstanceID string) (bool, error)
	UnRegisterMicroServiceInstance(microServiceID, microServiceInstanceID string) error
	UpdateMicroServiceInstanceStatus(microServiceID, microServiceInstanceID, status string) error
	UpdateMicroServiceProperties(microServiceID string, properties map[string]string) error
	UpdateMicroServiceInstanceProperties(microServiceID, microServiceInstanceID string, properties map[string]string) error
	AddSchemas(microServiceID, schemaName, schemaInfo string) error
}

// 开启服务注册
func enableRegistrator(opts Options) error {
	if config.GetRegistratorDisable() {
		return nil
	}

	rt := config.GetRegistratorType()
	if rt == "" {
		rt = DefaultRegistratorPlugin
	}

	// 创建register
	var err error
	DefaultRegistrator, err = NewRegistrator(rt, opts)
	if err != nil {
		return err
	}

	// 注册服务
	if err := RegisterService(); err != nil {
		openlog.Error(fmt.Sprintf("start backoff for register microservice: %s", err))
		startBackOff(RegisterService) // 同步 失败重试
	}

	openlog.Info(fmt.Sprintf("enable [%s] registrator.", rt))
	return nil
}

// InstallRegistrator install registrator plugin
func InstallRegistrator(name string, f func(opts Options) Registrator) {
	registryFunc[name] = f
	openlog.Info("Installed registry plugin: " + name)
}

//NewRegistrator return registrator
func NewRegistrator(name string, opts Options) (Registrator, error) {
	f := registryFunc[name]
	if f == nil {
		return nil, fmt.Errorf("no registry plugin: %s", name)
	}
	return f(opts), nil
}

// 获取注册中心配置
func getSpecifiedOptions() (oR, oSD, oCD Options, err error) {
	// 注册
	hostsR, schemeR, err := URIs2Hosts(strings.Split(config.GetRegistratorAddress(), ","))
	if err != nil {
		return
	}
	oR.Addrs = hostsR
	oR.Version = config.GetRegistratorAPIVersion()
	oR.TLSConfig, err = getTLSConfig(schemeR, RTag)
	if err != nil {
		return
	}
	if oR.TLSConfig != nil {
		oR.EnableSSL = true
	}

	// 发现
	hostsSD, schemeSD, err := URIs2Hosts(strings.Split(config.GetServiceDiscoveryAddress(), ","))
	if err != nil {
		return
	}
	oSD.Addrs = hostsSD
	oSD.Version = config.GetServiceDiscoveryAPIVersion()
	oSD.ConfigPath = config.GetServiceDiscoveryConfigPath()
	oSD.TLSConfig, err = getTLSConfig(schemeSD, SDTag)
	if err != nil {
		return
	}
	if oSD.TLSConfig != nil {
		oSD.EnableSSL = true
	}

	hostsCD, schemeCD, err := URIs2Hosts(strings.Split(config.GetContractDiscoveryAddress(), ","))
	if err != nil {
		return
	}
	oCD.Addrs = hostsCD
	oCD.Version = config.GetContractDiscoveryAPIVersion()
	oCD.TLSConfig, err = getTLSConfig(schemeCD, CDTag)
	if err != nil {
		return
	}
	if oCD.TLSConfig != nil {
		oCD.EnableSSL = true
	}
	return
}

// Enable create DefaultRegistrator
// 开启注册中心,初始化链接
func Enable() (err error) {
	mu.Lock()
	defer mu.Unlock()
	if IsEnabled { // 已开启
		return
	}

	// 获取各个链接配置
	var oR, oSD, oCD Options
	if oR, oSD, oCD, err = getSpecifiedOptions(); err != nil {
		return err
	}

	// 开启cache
	EnableRegistryCache()
	if err := enableRegistrator(oR); err != nil {
		return err
	}
	if err := enableServiceDiscovery(oSD); err != nil {
		return err
	}
	enableContractDiscovery(oCD)

	openlog.Info("Enabled Registry")
	IsEnabled = true
	return nil
}

// DoRegister for registering micro-service instances
func DoRegister() error {
	var (
		isAutoRegister bool
		t              = config.GetRegistratorAutoRegister()
	)
	switch t {
	case "":
		isAutoRegister = true
	case Auto:
		isAutoRegister = true
	case Manual:
		isAutoRegister = false
	default:
		{
			tmpErr := fmt.Errorf("parameter incorrect, autoregister: %s", t)
			openlog.Error(tmpErr.Error())
			return tmpErr
		}
	}
	if isAutoRegister {
		if err := RegisterServiceInstances(); err != nil {
			openlog.Error(fmt.Sprintf("start back off for register microservice instances background: %s", err))
			go startBackOff(RegisterServiceInstances)
		}
	}
	go HBService.Start()
	return nil
}
