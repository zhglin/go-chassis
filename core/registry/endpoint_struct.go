package registry

import (
	"strings"
)

//const
const (
	SSLEnabledQuery = "sslEnabled=true"
)

// Endpoint struct having full info about micro-service instance endpoint
// instance的Endpoint解析过来的
type Endpoint struct {
	SSLEnabled bool   `json:"sslEnabled"`
	Address    string `json:"address"`
}

// NewEndPoint return a Endpoint object what parse from url
// 新建
func NewEndPoint(schema string) (*Endpoint, error) {
	return parseAddress(schema)
}

//GenEndpoint return the endpoint string which it contain the sslEnabled=true query arg or not
func (e *Endpoint) GenEndpoint() string {
	if e.SSLEnabled {
		return e.Address + "?" + SSLEnabledQuery
	}
	return e.Address
}

//IsSSLEnable return it is use ssl or not
func (e *Endpoint) IsSSLEnable() bool {
	return e.SSLEnabled
}

//SetSSLEnable set ssl enable or not
func (e *Endpoint) SetSSLEnable(enabled bool) {
	e.SSLEnabled = enabled
}

func (e *Endpoint) String() string {
	return e.GenEndpoint()
}

// 解析address ssl
func parseAddress(address string) (*Endpoint, error) {
	ep := Endpoint{}
	idx := strings.Index(address, "?")
	// 设置了ssl
	if idx != -1 {
		if strings.Contains(address, SSLEnabledQuery) {
			ep.SSLEnabled = true
		}
		address = address[:idx]
	}
	if pIdx := strings.Index(address, ":"); pIdx == -1 {
		ep.Address = address
		return &ep, nil
	}
	ep.Address = address
	return &ep, nil
}
