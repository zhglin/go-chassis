package servicecomb

import (
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/v2/resilience/retry"
	"reflect"
	"strings"

	"github.com/go-chassis/go-chassis/v2/control"
	"github.com/go-chassis/go-chassis/v2/core/client"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/config/model"
	"github.com/go-chassis/go-chassis/v2/core/loadbalancer"
	"github.com/go-chassis/go-chassis/v2/third_party/forked/afex/hystrix-go/hystrix"
	"github.com/go-chassis/openlog"
)

//SaveToLBCache save configs
// 设置loadBalance配置的cache
func SaveToLBCache(raw *model.LoadBalancing) {
	openlog.Debug("Loading lb config from archaius into cache")
	oldKeys := LBConfigCache.Items()
	newKeys := make(map[string]bool)
	// if there is no config, none key will be updated  更新最新的cache
	if raw != nil {
		newKeys = reloadLBCache(raw)
	}
	// remove outdated keys  删除已经不存在的cache key
	for old := range oldKeys {
		if _, ok := newKeys[old]; !ok {
			LBConfigCache.Delete(old)
		}
	}

}

// 所有service默认的LB配置 返回对应的cache key
func saveDefaultLB(raw *model.LoadBalancing) string { // return updated key
	c := control.LoadBalancingConfig{
		Strategy:                raw.Strategy["name"],
		RetryEnabled:            raw.RetryEnabled,
		RetryOnSame:             raw.RetryOnSame,
		RetryOnNext:             raw.RetryOnNext,
		BackOffKind:             raw.Backoff.Kind,
		BackOffMin:              raw.Backoff.MinMs,
		BackOffMax:              raw.Backoff.MaxMs,
		SessionTimeoutInSeconds: raw.SessionStickinessRule.SessionTimeoutInSeconds,
		SuccessiveFailedTimes:   raw.SessionStickinessRule.SuccessiveFailedTimes,
	}

	setDefaultLBValue(&c)
	LBConfigCache.Set("", c, 0)
	return ""
}

// 设置每个依赖服务的balance配置 cache key=服务名
func saveEachLB(k string, raw model.LoadBalancingSpec) string { // return updated key
	c := control.LoadBalancingConfig{
		Strategy:                raw.Strategy["name"],
		RetryEnabled:            raw.RetryEnabled,
		RetryOnSame:             raw.RetryOnSame,
		RetryOnNext:             raw.RetryOnNext,
		BackOffKind:             raw.Backoff.Kind,
		BackOffMin:              raw.Backoff.MinMs,
		BackOffMax:              raw.Backoff.MaxMs,
		SessionTimeoutInSeconds: raw.SessionStickinessRule.SessionTimeoutInSeconds,
		SuccessiveFailedTimes:   raw.SessionStickinessRule.SuccessiveFailedTimes,
	}
	openlog.Info(fmt.Sprintf("save lb config [%s] [%v]", k, raw))
	setDefaultLBValue(&c)
	LBConfigCache.Set(k, c, 0)
	return k
}

// 设置默认的 负载均衡算法 重试策略
func setDefaultLBValue(c *control.LoadBalancingConfig) {
	if c.Strategy == "" {
		c.Strategy = loadbalancer.StrategyRoundRobin
	}
	if c.BackOffKind == "" {
		c.BackOffKind = retry.DefaultBackOffKind
	}
}

//SaveToCBCache save configs
// 设置hystrix缓存配置
func SaveToCBCache(raw *model.HystrixConfig) {
	openlog.Debug("Loading cb config from archaius into cache")
	oldKeys := CBConfigCache.Items()
	newKeys := make(map[string]bool)
	// if there is no config, none key will be updated
	if raw != nil {
		// 设置连接超时时间
		client.SetTimeoutToClientCache(raw.IsolationProperties)
		// 设置cache
		newKeys = reloadCBCache(raw)
	}
	// remove outdated keys 删除不存在的
	for old := range oldKeys {
		if _, ok := newKeys[old]; !ok {
			CBConfigCache.Delete(old)
		}
	}
}

// 更新设置hystrix的cache return cacheKey
func saveEachCB(serviceName, serviceType string) string { //return updated key
	command := serviceType
	if serviceName != "" {
		command = strings.Join([]string{serviceType, serviceName}, ".")
	}
	c := hystrix.CommandConfig{
		ForceFallback:          config.GetForceFallback(serviceName, serviceType),
		MaxConcurrentRequests:  config.GetMaxConcurrentRequests(command, serviceType),
		ErrorPercentThreshold:  config.GetErrorPercentThreshold(command, serviceType),
		RequestVolumeThreshold: config.GetRequestVolumeThreshold(command, serviceType),
		SleepWindow:            config.GetSleepWindow(command, serviceType),
		ForceClose:             config.GetForceClose(serviceName, serviceType),
		ForceOpen:              config.GetForceOpen(serviceName, serviceType),
		CircuitBreakerEnabled:  config.GetCircuitBreakerEnabled(command, serviceType),
	}
	cbcCacheKey := GetCBCacheKey(serviceName, serviceType)
	cbcCacheValue, b := CBConfigCache.Get(cbcCacheKey)
	formatString := "save circuit breaker config [%#v] for [%s] "
	// 不存在 设置cache
	if !b || cbcCacheValue == nil {
		openlog.Info(fmt.Sprintf(formatString, c, serviceName))
		CBConfigCache.Set(cbcCacheKey, c, 0)
		return cbcCacheKey
	}

	// 已存在 类型不对 重新设置
	commandConfig, ok := cbcCacheValue.(hystrix.CommandConfig)
	if !ok {
		openlog.Info(fmt.Sprintf(formatString, c, serviceName))
		CBConfigCache.Set(cbcCacheKey, c, 0)
		return cbcCacheKey
	}

	// 没变化
	if c == commandConfig {
		return cbcCacheKey
	}

	// 已修改
	openlog.Info(fmt.Sprintf(formatString, c, serviceName))
	CBConfigCache.Set(cbcCacheKey, c, 0)
	return cbcCacheKey
}

//GetCBCacheKey generate cache key
// hystrix缓存key
func GetCBCacheKey(serviceName, serviceType string) string {
	key := serviceType
	if serviceName != "" {
		key = serviceType + ":" + serviceName
	}
	return key
}

// 更新src最新的配置 进行cache的更新
func reloadLBCache(src *model.LoadBalancing) map[string]bool { //return updated keys
	keys := make(map[string]bool)
	k := saveDefaultLB(src) // 全局统一的配置
	keys[k] = true
	if src.AnyService == nil {
		return keys
	}
	// 每个service独立的配置
	for name, conf := range src.AnyService {
		k = saveEachLB(name, conf)
		keys[k] = true
	}
	return keys
}

// 解析hystrix配置 并重置cache
func reloadCBCache(src *model.HystrixConfig) map[string]bool { //return updated keys
	keys := make(map[string]bool)
	// global level config 	// consumer的全局默认配置
	k := saveEachCB("", common.Consumer)
	keys[k] = true

	// provider的全局默认配置
	k = saveEachCB("", common.Provider)
	keys[k] = true
	// get all services who have configs
	consumers := make([]string, 0)
	providers := make([]string, 0)
	consumerMap := map[string]bool{}
	providerMap := map[string]bool{}

	// if a service has configurations of IsolationProperties|
	// CircuitBreakerProperties|FallbackPolicyProperties|FallbackProperties,
	// it's configuration should be added to cache when framework starts
	// 获取所有service
	for _, p := range []interface{}{
		src.IsolationProperties,
		src.CircuitBreakerProperties,
		src.FallbackProperties,
		config.GetHystrixConfig().FallbackPolicyProperties} {
		if services, err := getServiceNamesByServiceTypeAndAnyService(p, common.Consumer); err != nil {
			openlog.Error(fmt.Sprintf("Parse services from config failed: %v", err.Error()))
		} else {
			consumers = append(consumers, services...)
		}
		if services, err := getServiceNamesByServiceTypeAndAnyService(p, common.Provider); err != nil {
			openlog.Error(fmt.Sprintf("Parse services from config failed: %v", err.Error()))
		} else {
			providers = append(providers, services...)
		}
	}
	// remove duplicate service names 去重
	for _, name := range consumers {
		consumerMap[name] = true
	}
	for _, name := range providers {
		providerMap[name] = true
	}
	// service level config 生成各个service的配置
	for name := range consumerMap {
		k = saveEachCB(name, common.Consumer)
		keys[k] = true
	}
	for name := range providerMap {
		k = saveEachCB(name, common.Provider)
		keys[k] = true
	}
	return keys
}

func getServiceNamesByServiceTypeAndAnyService(i interface{}, serviceType string) (services []string, err error) {
	// check type
	tmpType := reflect.TypeOf(i)
	if tmpType.Kind() != reflect.Ptr {
		return nil, errors.New("input must be an ptr")
	}
	// check value
	tmpValue := reflect.ValueOf(i)
	if !tmpValue.IsValid() {
		return []string{}, nil
	}

	inType := tmpType.Elem()
	propertyName := inType.Name()

	formatFieldNotExist := "field %s not exist"
	formatFieldNotExpected := "field %s is not type %s"
	// check type
	tmpFieldType, ok := inType.FieldByName(serviceType)
	if !ok {
		return nil, fmt.Errorf(formatFieldNotExist, propertyName+"."+serviceType)
	}
	if tmpFieldType.Type.Kind() != reflect.Ptr {
		return nil, fmt.Errorf(formatFieldNotExpected, propertyName+"."+serviceType, reflect.Ptr)
	}
	// check value
	inValue := reflect.Indirect(tmpValue)
	tmpFieldValue := inValue.FieldByName(serviceType)
	if !tmpFieldValue.IsValid() {
		return []string{}, nil
	}

	anyServiceFieldName := "AnyService"
	//check type
	fieldType := tmpFieldType.Type.Elem()
	tmpAnyServiceFieldType, ok := fieldType.FieldByName(anyServiceFieldName)
	if !ok {
		return nil, fmt.Errorf(formatFieldNotExist, propertyName+"."+serviceType+"."+anyServiceFieldName)
	}
	if tmpAnyServiceFieldType.Type.Kind() != reflect.Map {
		return nil, fmt.Errorf(formatFieldNotExpected, propertyName+"."+serviceType+"."+anyServiceFieldName, reflect.Map)
	}
	// check value
	fieldValue := reflect.Indirect(tmpFieldValue)
	anyServiceFieldValue := fieldValue.FieldByName(anyServiceFieldName)
	if !anyServiceFieldValue.IsValid() {
		return []string{}, nil
	}

	// get service names
	names := anyServiceFieldValue.MapKeys()
	services = make([]string, 0)
	for _, name := range names {
		if name.Kind() != reflect.String {
			return nil, fmt.Errorf(formatFieldNotExpected, "key of "+propertyName+"."+serviceType+"."+anyServiceFieldName, reflect.String)
		}
		services = append(services, name.String())
	}
	return services, nil
}
