package config

import (
	stringutil "github.com/go-chassis/go-chassis/v2/pkg/string"
	"gopkg.in/yaml.v2"
)

//OneServiceRule save route rule for one service
// 一个服务的路由规则
type OneServiceRule []*RouteRule

//Len return the length of rule
func (o OneServiceRule) Len() int {
	return len(o)
}

//Value return the rule
func (o OneServiceRule) Value() []*RouteRule {
	return o
}

//NewServiceRule create a rule by raw data
// 解析rule
func NewServiceRule(raw string) (*OneServiceRule, error) {
	b := stringutil.Str2bytes(raw)
	r := &OneServiceRule{}
	err := yaml.Unmarshal(b, r)
	return r, err
}

//ServiceComb hold all config items
type ServiceComb struct {
	Prefix Prefix `yaml:"servicecomb"`
}

//Prefix hold all config items
type Prefix struct {
	RouteRule       map[string]string `yaml:"routeRule"`      //service name is key,value is route rule yaml config
	SourceTemplates map[string]string `yaml:"sourceTemplate"` //template name is key, value is template policy
}

// Router define where rule comes from
type Router struct {
	Infra   string `yaml:"infra"`
	Address string `yaml:"address"`
}

// RouteRule is having route rule parameters
type RouteRule struct {
	Precedence int         `json:"precedence" yaml:"precedence"` // 优先级 根据此字段进行排序
	Routes     []*RouteTag `json:"route" yaml:"route"`           // 权重不同的多个tag
	Match      Match       `json:"match" yaml:"match"`           // 匹配规则
}

// RouteTag gives route tag information
type RouteTag struct {
	Tags   map[string]string `json:"tags" yaml:"tags"`     // 对应的标签
	Weight int               `json:"weight" yaml:"weight"` // 权重
	Label  string            // tags转换 k:v|k:v
}

// Match is checking source, source tags, and http headers
type Match struct {
	Refer       string                       `json:"refer" yaml:"refer"`             // 是否使用已经定义好的match规则
	Source      string                       `json:"source" yaml:"source"`           // invocation.SourceMicroService 来源服务
	SourceTags  map[string]string            `json:"sourceTags" yaml:"sourceTags"`   // 请求的tag invocation.metadata
	HTTPHeaders map[string]map[string]string `json:"httpHeaders" yaml:"httpHeaders"` // http header
	Headers     map[string]map[string]string `json:"headers" yaml:"headers"`         // caseInsensitive 可以设置不区分大小写
}

//DarkLaunchRule dark launch rule
//Deprecated
type DarkLaunchRule struct {
	Type  string      `json:"policyType"` // RULE/RATE
	Items []*RuleItem `json:"ruleItems"`
}

//RuleItem rule item
//Deprecated
type RuleItem struct {
	GroupName       string   `json:"groupName"`
	GroupCondition  string   `json:"groupCondition"`  // version=0.0.1
	PolicyCondition string   `json:"policyCondition"` // 80/test!=2
	CaseInsensitive bool     `json:"caseInsensitive"`
	Versions        []string `json:"versions"`
}

//MatchPolicy specify a request mach policy
type MatchPolicies struct {
	Matches []MatchPolicy `yaml:"matches"`
}

//MatchPolicy specify a request mach policy
type MatchPolicy struct {
	TrafficMarkPolicy string                       `yaml:"trafficMarkPolicy"` // 是否整个链路都是此标记
	Headers           map[string]map[string]string `yaml:"headers"`           // 请求头  [headTitle][比较规则][value] 有一个不匹配就失败
	APIPaths          map[string]string            `yaml:"apiPath"`           // [比较][value]  匹配成功一个就成功
	Method            []string                     `yaml:"method"`            // [GET,POST] 匹配成功一个就成功
}

//LimiterConfig is rate limiter policy
type LimiterConfig struct {
	Match string
	QPS   string
}
