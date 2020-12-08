package client

import (
	"fmt"
	"github.com/go-chassis/openlog"
)

// NewFunc is function for the client
// 创建链接的函数
type NewFunc func(Options) (ProtocolClient, error)

// 不同协议对应的链接创建函数
var plugins = make(map[string]NewFunc)

// GetClientNewFunc is to get the client
// 获取指定协议的链接创建函数
func GetClientNewFunc(name string) (NewFunc, error) {
	f := plugins[name]
	if f == nil {
		return nil, fmt.Errorf("don't have client plugin %s", name)
	}
	return f, nil
}

// InstallPlugin is plugin for the new function
// 注册指定协议的链接创建函数
func InstallPlugin(protocol string, f NewFunc) {
	openlog.Info("Install client plugin, protocol: " + protocol)
	plugins[protocol] = f
}
