package client

import (
	"github.com/go-chassis/openlog"
	"net/url"
)

// 格式化instance的Endpoints  scheme://host
func getProtocolMap(eps []string) map[string]string {
	m := make(map[string]string)
	for _, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			openlog.Error("url err: " + err.Error())
			continue
		}
		m[u.Scheme] = u.Host
	}
	return m
}
