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

package basicauth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/go-chassis/go-chassis/v2/core/handler"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
	"github.com/go-chassis/go-chassis/v2/core/status"
	"github.com/go-chassis/openlog"
	"net/http"
	"strings"
)

//errors
var (
	ErrInvalidBase64 = errors.New("invalid base64")
	ErrNoHeader      = errors.New("not authorized")
	ErrInvalidAuth   = errors.New("invalid authentication")
)

//HeaderAuth is common auth header
const HeaderAuth = "Authorization"

//Handler is is a basic auth pre process raw data in handler
type Handler struct {
}

// Handle pre process raw data in handler
func (ph *Handler) Handle(chain *handler.Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	// 非http协议
	var req *http.Request
	if r, ok := i.Args.(*http.Request); ok {
		req = r
	} else if r, ok := i.Args.(*restful.Request); ok {
		req = r.Request
	} else {
		openlog.Error(fmt.Sprintf("this handler only works for http protocol, wrong type: %t", i.Args))
		return
	}

	// header头
	subject := req.Header.Get(HeaderAuth)
	if subject == "" {
		handler.WriteBackErr(ErrNoHeader, status.Status(i.Protocol, status.Unauthorized), cb)
		return
	}

	// 解析账号密码
	u, p, err := decode(subject)
	if err != nil {
		openlog.Error("can not decode base 64:" + err.Error())
		handler.WriteBackErr(ErrNoHeader, status.Status(i.Protocol, status.Unauthorized), cb)
		return
	}

	// 账号密码校验
	err = auth.Authenticate(u, p)
	if err != nil {
		handler.WriteBackErr(ErrNoHeader, status.Status(i.Protocol, status.Unauthorized), cb)
		return
	}

	// 授权校验
	if auth.Authorize != nil {
		err = auth.Authorize(u, req)
		if err != nil {
			handler.WriteBackErr(ErrNoHeader, status.Status(i.Protocol, status.Unauthorized), cb)
			return
		}
	}
	chain.Next(i, cb)
}

// 基本认证
func newBasicAuth() handler.Handler {
	return &Handler{}
}

// Name returns the router string
func (ph *Handler) Name() string {
	return "basicAuth"
}

// base64解析账号 密码
func decode(subject string) (user string, pwd string, err error) {
	parts := strings.Split(subject, " ")
	if len(parts) != 2 {
		return "", "", ErrInvalidAuth

	}
	if parts[0] != "Basic" {
		return "", "", ErrInvalidAuth
	}
	s, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", ErrInvalidBase64
	}

	result := strings.Split(string(s), ":")
	if len(result) != 2 {
		return "", "", ErrInvalidAuth
	}

	return result[0], result[1], nil
}
