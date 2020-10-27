package registry

import (
	"strings"

	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/pkg/runtime"
	"github.com/go-chassis/openlog"
	"github.com/patrickmn/go-cache"
)

const (
	//DefaultExpireTime default expiry time is kept as 0
	DefaultExpireTime = 0
)

//MicroserviceInstanceIndex key: ServiceName, value: []instance
// service对应的instance serviceName => instance
var MicroserviceInstanceIndex CacheIndex

//ipIndexedCache is for caching map of instance IP and service information
//key: instance ip, value: SourceInfo
var ipIndexedCache *cache.Cache

//SchemaInterfaceIndexedCache key: schema interface name value: []*microservice
var SchemaInterfaceIndexedCache *cache.Cache

//SchemaServiceIndexedCache key: schema service name value: []*microservice
var SchemaServiceIndexedCache *cache.Cache

// ProvidersMicroServiceCache  key: micro service  name and appId, value: []*MicroService
// 依赖service的cache  key=serviceName|appId  MicroService{ServiceName: serverName, AppID: appID}
var ProvidersMicroServiceCache *cache.Cache

func initCache() *cache.Cache { return cache.New(DefaultExpireTime, 0) }

//EnableRegistryCache init caches
// 初始化各个cache
func EnableRegistryCache() {
	MicroserviceInstanceIndex = NewIndexCache()
	ipIndexedCache = initCache()
	SchemaServiceIndexedCache = initCache()
	SchemaInterfaceIndexedCache = initCache()
	ProvidersMicroServiceCache = initCache()
}

// CacheIndex is a unified local instances cache manager
type CacheIndex interface {
	Get(service string, tags map[string]string) ([]*MicroServiceInstance, bool)
	//Set will overwrite all instances correspond to a service name
	Set(service string, instances []*MicroServiceInstance)
	FullCache() *cache.Cache
	Delete(service string)
}

//SetIPIndex save ip index
func SetIPIndex(ip string, si *SourceInfo) {
	ipIndexedCache.Set(ip, si, 0)
}

//GetIPIndex get ip corresponding source info
func GetIPIndex(ip string) *SourceInfo {
	cacheDatum, ok := ipIndexedCache.Get(ip)
	if !ok {
		return nil
	}
	si, ok := cacheDatum.(*SourceInfo)
	if !ok {
		return nil
	}
	return si
}

// GetProvidersFromCache get local provider simpleCache
// 获取依赖service
func GetProvidersFromCache() []*MicroService {
	microServices := make([]*MicroService, 0)
	items := ProvidersMicroServiceCache.Items()
	for _, item := range items {
		microService, ok := item.Object.(MicroService)
		if !ok {
			openlog.Warn("not microService type")
			continue
		}
		microService.Version = common.AllVersion // 设置版本规则
		microServices = append(microServices, &microService)
	}
	return microServices
}

// AddProviderToCache refresh provider simpleCache
// 添加依赖service
func AddProviderToCache(serverName, appID string) {
	if appID == "" {
		appID = runtime.App
		if appID == "" {
			appID = common.DefaultApp
		}
	}
	key := strings.Join([]string{serverName, appID}, "|")
	if _, ok := ProvidersMicroServiceCache.Get(key); !ok {
		ProvidersMicroServiceCache.Set(key, MicroService{ServiceName: serverName, AppID: appID}, 0)
	}
}
