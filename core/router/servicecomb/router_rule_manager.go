package servicecomb

import (
	"errors"
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/openlog"
	"strings"
)

// constant for route rule keys
const (
	DarkLaunchKey      = "^servicecomb\\.darklaunch\\.policy\\."
	DarkLaunchKeyV2    = "^servicecomb\\.routeRule\\."
	DarkLaunchPrefix   = "servicecomb.darklaunch.policy."
	DarkLaunchPrefixV2 = "servicecomb.routeRule."
	DarkLaunchTypeRule = "RULE"
	DarkLaunchTypeRate = "RATE"
)

/*
servicecomb:
    routeRule:
      {targetServiceName}: |# 服务名
        - precedence: {number} #优先级
          match:        #匹配策略
            source: {sourceServiceName} #匹配某个服务名
            headers:          #header匹配
              {key0}:
                regex: {regex}
                caseInsensitive: false # 是否区分大小写，默认为false，区分大小写
              {key1}
                exact: {=？}
          route: #路由规则
            - weight: {percent} #权重值
              tags:
                version: {version1}
                app: {appId}
        - precedence: {number1}
          match:
            refer: {matchPolicy} #参考某个match policy
          route:
            - weight: {percent}
              tags:
                version: {version2}
                app: {appId}
*/

//MergeLocalAndRemoteConfig get router config from archaius,
//including local file,memory and config server
// 从配置中心获取rule配置
func MergeLocalAndRemoteConfig() (map[string][]*config.RouteRule, error) {
	destinations := make(map[string][]*config.RouteRule)
	//then get config from archaius and simply overwrite rule from file
	ruleV1Map := make(map[string]interface{})
	ruleV2Map := make(map[string]interface{})
	// 配置中心获取所有配置
	configMap := archaius.GetConfigs()
	//filter out key:value pairs which are not route rules
	prepareRule(configMap, ruleV1Map, ruleV2Map)
	rules, e := processV1Rule(ruleV1Map, destinations)
	if e != nil {
		return rules, e
	}
	routeRules, e := processV2Rule(ruleV2Map, destinations)
	if e != nil {
		return routeRules, e
	}
	return destinations, nil
}

// 解析v2 rule
func processV2Rule(ruleV2Map map[string]interface{}, destinations map[string][]*config.RouteRule) (map[string][]*config.RouteRule, error) {
	for k, v := range ruleV2Map {
		value, ok := v.(string)
		if !ok {
			return nil, errors.New("route rule is not a yaml string format, please check the configuration in config server")
		}

		service := strings.Replace(k, DarkLaunchPrefixV2, "", 1)
		r, err := config.NewServiceRule(value)
		if err != nil {
			openlog.Error("convert failed: " + err.Error())
		}
		destinations[service] = r.Value()
	}
	return nil, nil
}

func processV1Rule(ruleV1Map map[string]interface{}, destinations map[string][]*config.RouteRule) (map[string][]*config.RouteRule, error) {
	for k, v := range ruleV1Map {
		value, ok := v.(string)
		if !ok {
			return nil, errors.New("route rule is not a json string format, please check the configuration in config server")
		}

		service := strings.Replace(k, DarkLaunchPrefix, "", 1)
		r, err := ConvertJSON2RouteRule(value)
		if err != nil {
			openlog.Error("convert failed: " + err.Error())
		}
		destinations[service] = r
	}
	return nil, nil
}

// 从配置中心过滤出来router配置
func prepareRule(configMap map[string]interface{}, ruleV1Map map[string]interface{}, ruleV2Map map[string]interface{}) {
	for k, v := range configMap {
		if strings.HasPrefix(k, DarkLaunchPrefix) {
			ruleV1Map[k] = v
			continue
		}
		if strings.HasPrefix(k, DarkLaunchPrefixV2) {
			openlog.Debug("get one route rule:" + k)
			ruleV2Map[k] = v
			continue
		}
	}
}
