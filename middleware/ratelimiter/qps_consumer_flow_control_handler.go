package ratelimiter

import (
	"errors"
	"github.com/go-chassis/go-chassis/v2/resilience/rate"
	"github.com/go-chassis/openlog"
	"net/http"

	"github.com/go-chassis/go-chassis/v2/control"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/handler"
	"github.com/go-chassis/go-chassis/v2/core/invocation"
)

// names
const (
	Consumer = "ratelimiter-consumer"
	Provider = "ratelimiter-provider"
	Name     = "rate-limiter"
)

// ConsumerRateLimiterHandler consumer rate limiter handler
// consumer端的依赖的各个服务的限流
type ConsumerRateLimiterHandler struct{}

// Handle is handles the consumer rate limiter APIs
func (rl *ConsumerRateLimiterHandler) Handle(chain *handler.Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	// 获取配置
	rlc := control.DefaultPanel.GetRateLimiting(*i, common.Consumer)
	if !rlc.Enabled {
		chain.Next(i, cb)

		return
	}
	//qps rate <=0
	if rlc.Rate <= 0 {
		r := newErrResponse(i)
		cb(r)
		return
	}
	//get operation meta info ms.schema, ms.schema.operation, ms
	// rlc.key用于区分不同的rateLimiters
	if rate.GetRateLimiters().TryAccept(rlc.Key, rlc.Rate, rlc.Rate/5) {
		chain.Next(i, cb)
	} else {
		r := newErrResponse(i)
		cb(r)
		return
	}

}

func newErrResponse(i *invocation.Invocation) *invocation.Response {
	switch resp := i.Reply.(type) {
	case *http.Response:
		resp.StatusCode = http.StatusTooManyRequests
	}
	r := &invocation.Response{}
	r.Status = http.StatusTooManyRequests
	r.Err = errors.New("too many requests")
	return r
}

func newConsumerRateLimiterHandler() handler.Handler {
	return &ConsumerRateLimiterHandler{}
}

// Name returns name
func (rl *ConsumerRateLimiterHandler) Name() string {
	return "ratelimiter-consumer"
}

// 针对service的精确化流控
func init() {
	err := handler.RegisterHandler(Consumer, newConsumerRateLimiterHandler)
	if err != nil {
		openlog.Error(err.Error())
	}
	err = handler.RegisterHandler(Provider, newProviderRateLimiterHandler)
	if err != nil {
		openlog.Error(err.Error())
	}
}
