package registry

import (
	"crypto/tls"
	"time"
)

// Options having micro-service parameters
type Options struct {
	Addrs      []string  // 链接地址
	EnableSSL  bool			// 是否开启tls
	Timeout    time.Duration
	TLSConfig  *tls.Config   //tls配置
	Compressed bool
	Verbose    bool
	Version    string  // register api 版本
	ConfigPath string
}
