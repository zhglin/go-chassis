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

package marker

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/openlog"
	"gopkg.in/yaml.v2"
	"net/http"
	"strings"
	"sync"
)

const (
	Once       = "once"
	PerService = "perService"
)

// 保存标记信息
var matches sync.Map

//Operate decide value match expression or not
type Operate func(value, expression string) bool

// 支持的比较方式
var operatorPlugin = map[string]Operate{
	"exact":     exact,     // 精确比较
	"contains":  contains,  // 子串
	"regex":     regex,     // 正则匹配
	"noEqu":     noEqu,     // 不等于
	"less":      less,      // 小于
	"noLess":    noLess,    // 大于等于
	"greater":   greater,   // 大于
	"noGreater": noGreater, // 小于等于
}

//Install a strategy
func Install(name string, m Operate) {
	operatorPlugin[name] = m
}

//Mark mark an invocation with matchName by match policy
// 进行match匹配
func Mark(inv *invocation.Invocation) {
	matchName := ""
	policy := "once"
	// 从所有match中找个匹配的
	matches.Range(func(k, v interface{}) bool {
		mps, ok := v.(*config.MatchPolicies)
		if !ok {
			return true
		}
		for _, mp := range mps.Matches {
			if isMatch(inv, &mp) {
				// 匹配成功
				if name, ok := k.(string); ok {
					matchName = name
					policy = mp.TrafficMarkPolicy
					return false
				}
			}
		}

		return true
	})

	// 匹配上
	if matchName != "" {
		//the invocation math policy
		if policy == Once {
			inv.SetHeader(common.HeaderMark, matchName)
		}
		inv.Mark(matchName)
	}
}

// 是否匹配上matchPolicy  有一个条件匹配失败就false
func isMatch(inv *invocation.Invocation, matchPolicy *config.MatchPolicy) bool {
	// header未匹配上
	if !headsMatch(inv.Headers(), matchPolicy.Headers) {
		return false
	}

	// 非httpRequest false
	var req *http.Request
	switch r := inv.Args.(type) {
	case *http.Request:
		req = r
	case *restful.Request:
		req = r.Request
	default:
		return false
	}

	// httpPath
	if len(matchPolicy.APIPaths) != 0 && !apiMatch(req.URL.Path, matchPolicy.APIPaths) {
		return false
	}

	// httpMethod
	if len(matchPolicy.Method) != 0 {
		if !methodMatch(req.Method, matchPolicy.Method) {
			return false
		}
	}
	return true
}

// http method匹配
func methodMatch(reqMethod string, methods []string) bool {
	matchMethod := false
	for _, m := range methods {
		if strings.ToUpper(reqMethod) == m {
			matchMethod = true
		}
	}
	return matchMethod
}

// http path匹配
func apiMatch(apiPath string, apiPolicy map[string]string) bool {
	if len(apiPolicy) == 0 {
		return true
	}

	// 多个 匹配一个就算成功
	for strategy, exp := range apiPolicy {
		if ok, _ := Match(strategy, apiPath, exp); ok {
			return true
		}
	}
	return false
}

// heads 请求头匹配
func headsMatch(headers map[string]string, headPolicy map[string]map[string]string) bool {
	for key, policy := range headPolicy {
		// header中的title
		val := headers[key]
		// 不存在
		if val == "" {
			return false
		}
		// 有一个不匹配就失败
		for strategy, exp := range policy {
			if o, err := Match(strategy, val, exp); err != nil || !o {
				return false
			}
		}
	}
	return true
}

//match compare value and expression
// 比较规则是否匹配
func Match(operator, value, expression string) (bool, error) {
	f, ok := operatorPlugin[operator]
	if !ok {
		return false, fmt.Errorf("invalid match method")
	}
	return f(value, expression), nil
}

//SaveMatchPolicy saves match policy
// 写入match规则
func SaveMatchPolicy(name, value string, k string) error {
	m := &config.MatchPolicies{}
	err := yaml.Unmarshal([]byte(value), m)
	if err != nil {
		openlog.Error("invalid policy " + k + ":" + err.Error())
		return err
	}
	openlog.Info("add match policy", openlog.WithTags(openlog.Tags{
		"module": "marker",
		"event":  "update",
	}))
	matches.Store(name, m)
	return nil
}

//Policy return policy
func Policy(name string) *config.MatchPolicies {
	i, ok := matches.Load(name)
	if !ok {
		return nil
	}
	m, ok := i.(*config.MatchPolicies)
	if !ok {
		return nil
	}
	return m
}
