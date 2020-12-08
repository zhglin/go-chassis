package client

import (
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var onceInit sync.Once
var onceMonitor sync.Once
var instance *AddressPool

const (
	available               string = "available"   // 可用
	unavailable             string = "unavailable" // 不可用
	defaultCheckSCIInterval        = 25            // default sc instance health check interval in second
)

// AddressPool registry address pool
// register的可用地址
type AddressPool struct {
	addressMap map[string]string // 所有获取的address
	status     map[string]string // 每个address的可用状态
	mutex      sync.RWMutex
}

// GetInstance Get registry pool instance
// 只是初始化了下
func GetInstance() *AddressPool {
	onceInit.Do(func() {
		instance = &AddressPool{
			addressMap: make(map[string]string),
			status:     make(map[string]string),
		}
	})
	return instance
}

// SetAddress set addresses to pool
// 设置register地址 配置里读取的
func (p *AddressPool) SetAddress(addresses []string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.addressMap = make(map[string]string)
	for _, v := range addresses {
		p.status[v] = available
		p.addressMap[v] = v
	}
}

// GetAvailableAddress Get an available address from pool by roundrobin
// 获取可用的address 每次使用不同的地址
func (p *AddressPool) GetAvailableAddress() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	addrs := make([]string, 0)
	for _, v := range p.addressMap {
		if p.status[v] == available {
			addrs = append(addrs, v)
		}
	}

	// 轮询获取地址
	next := RoundRobin(addrs)
	addr, err := next()
	if err != nil {
		return DefaultAddr
	}
	return addr
}

// 校验service center 地址是否可用
func (p *AddressPool) checkConnectivity() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	timeOut := time.Duration(1) * time.Second
	for _, v := range p.addressMap {
		conn, err := net.DialTimeout("tcp", v, timeOut) // 能否链接成功
		if err != nil {
			p.status[v] = unavailable
		} else {
			p.status[v] = available
			conn.Close()
		}
	}
}

//Monitor monitor each service center network connectivity
// 定时检查address
func (p *AddressPool) Monitor() {
	onceMonitor.Do(func() {
		p.checkConnectivity() // 首次检查address
		var interval time.Duration
		v, isExist := os.LookupEnv(EnvCheckSCIInterval) // 配置的间隔时间
		if !isExist {
			interval = defaultCheckSCIInterval
		} else {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				interval = defaultCheckSCIInterval
			} else {
				interval = time.Duration(i)
			}
		}
		ticker := time.NewTicker(interval * time.Second)
		quit := make(chan struct{})

		go func() {
			for {
				select {
				case <-ticker.C:
					p.checkConnectivity() // 定时检查
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	})
}
