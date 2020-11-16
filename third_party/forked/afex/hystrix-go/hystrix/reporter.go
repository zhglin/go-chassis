package hystrix

import (
	"errors"
	"github.com/go-chassis/openlog"
	"time"
)

//Reporter receive a circuit breaker Metrics and sink it to monitoring system
type Reporter func(cb *CircuitBreaker) error

//ErrDuplicated means you can not install reporter with same name
var ErrDuplicated = errors.New("duplicated reporter")
var reporterPlugins = make(map[string]Reporter) // 上报接口

//InstallReporter install reporter implementation
//it receives a circuit breaker and sink its Metrics to monitoring system
// 注册上报接口
func InstallReporter(name string, reporter Reporter) error {
	_, ok := reporterPlugins[name]
	if ok {
		return ErrDuplicated
	}
	reporterPlugins[name] = reporter
	openlog.Info("install reporter plugin:" + name)
	return nil
}

//StartReporter starts reporting to reporters
// 数据上报 需要业务调用开启
func StartReporter() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			circuitBreakersMutex.RLock()
			for _, cb := range circuitBreakers {
				for k, report := range reporterPlugins {
					openlog.Debug("report circuit metrics to " + k)
					if err := report(cb); err != nil {
						openlog.Warn("can not report: " + err.Error())
					}
				}
			}
			circuitBreakersMutex.RUnlock()
		}
	}
}
