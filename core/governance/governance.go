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
// 流量标记
package governance

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/openlog"
	"strings"
)

//prefix const
const (
	KindMatchPrefix        = "servicecomb.match"
	KindRateLimitingPrefix = "servicecomb.rateLimiting"
)

// 默认的解析函数
var processFuncMap = map[string]ProcessFunc{
	//build-in
	KindMatchPrefix:        ProcessMatch,   // 匹配
	KindRateLimitingPrefix: ProcessLimiter, // 限流
}

//ProcessFunc process a config
// 解析函数类型
type ProcessFunc func(key string, value string) error

//InstallProcessor install a func to process config,
//if a config key matches the key prefix, then the func will process the config
// 注册不同配置的解析函数
func InstallProcessor(keyPrefix string, process ProcessFunc) {
	processFuncMap[keyPrefix] = process
}

//Init go through all governance configs
//and call process func according to key prefix
// 初始化 读取 解析配置
func Init() {
	configMap := archaius.GetConfigs()
	openlog.Info("process all governance rules")
	for k, v := range configMap {
		value, ok := v.(string)
		if !ok {
			openlog.Warn("not string format,key:" + k)
		}
		openlog.Debug(k + ":" + value)
		// 解析
		for prefix, f := range processFuncMap {
			if strings.HasPrefix(k, prefix) {
				err := f(k, value)
				if err != nil {
					openlog.Error("can not process " + prefix + ":" + err.Error())
				}
				break
			}
		}
	}
}
