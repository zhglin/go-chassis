package hystrix

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/v2/third_party/forked/afex/hystrix-go/hystrix/metric_collector"
	"github.com/go-chassis/go-chassis/v2/third_party/forked/afex/hystrix-go/hystrix/rolling"
	"github.com/go-chassis/openlog"
)

// command执行结果
type commandExecution struct {
	Types       []string      `json:"types"`
	Start       time.Time     `json:"start_time"`
	RunDuration time.Duration `json:"run_duration"`
}

// command的结果转换器
type metricExchange struct {
	Name    string
	Updates chan *commandExecution // command的执行结果事件
	Mutex   *sync.RWMutex

	metricCollectors []metricCollector.MetricCollector // 数据收集器
}

// command的执行结果的转换汇总
func newMetricExchange(name string, num int) *metricExchange {
	m := &metricExchange{}
	m.Name = name

	m.Updates = make(chan *commandExecution, 2000)
	m.Mutex = &sync.RWMutex{}
	m.metricCollectors = metricCollector.Registry.InitializeMetricCollectors(name)
	m.Reset() // 重置

	// 开启多个协程进行事件消费
	for i := 0; i < num; i++ {
		go m.Monitor()
	}
	openlog.Debug(fmt.Sprintf(" launched [%d] Metrics consumer", num))
	return m
}

// The Default Collector function will panic if collectors are not setup to specification.
// 获取默认的MetricCollector
func (m *metricExchange) DefaultCollector() *metricCollector.DefaultMetricCollector {
	if len(m.metricCollectors) < 1 {
		panic("No Metric Collectors Registered")
	}
	collection, ok := m.metricCollectors[0].(*metricCollector.DefaultMetricCollector)
	if !ok {
		panic("Default metric collector is not registered correctly. The default metric collector must be registered first")
	}
	return collection
}

// 事件消费
func (m *metricExchange) Monitor() {
	for update := range m.Updates {
		// we only grab a read lock to make sure Reset() isn't changing the numbers.
		// 我们只获取一个读锁，以确保Reset()不会改变数字。
		m.Mutex.RLock()

		totalDuration := time.Since(update.Start) // 请求开始到结束的总的花费
		for _, collector := range m.metricCollectors {
			m.IncrementMetrics(collector, update, totalDuration) // 数据记录
		}

		m.Mutex.RUnlock()
	}
}

// 对执行结果进行数据记录
func (m *metricExchange) IncrementMetrics(collector metricCollector.MetricCollector, update *commandExecution, totalDuration time.Duration) {
	// granular Metrics
	if update.Types[0] == "success" { // run执行成功
		collector.IncrementAttempts()  // 总请求数
		collector.IncrementSuccesses() // 成功的请求数
	}
	if update.Types[0] == "failure" { // run 失败
		collector.IncrementFailures() // run失败的计数

		collector.IncrementAttempts() // 总请求数
		collector.IncrementErrors()   // 异常的请计数
	}
	if update.Types[0] == "rejected" { // 超过并发限制
		collector.IncrementRejects() // 超过并发限制计数

		collector.IncrementAttempts() // 总请求数
		collector.IncrementErrors()   // 异常的请计数
	}
	if update.Types[0] == "short-circuit" { // 被熔断
		collector.IncrementShortCircuits() // 被熔断计数

		collector.IncrementAttempts() // 总请求数
	}
	if update.Types[0] == "timeout" { // 超时的 (未使用)
		collector.IncrementTimeouts() // 超时的计数

		collector.IncrementAttempts() // 总请求数
		collector.IncrementErrors()   // 异常的请计数
	}

	// 执行了fallback
	if len(update.Types) > 1 {
		// fallback Metrics
		if update.Types[1] == "fallback-success" { // fallback执行成功
			collector.IncrementFallbackSuccesses()
		}
		if update.Types[1] == "fallback-failure" { // fallback执行失败
			collector.IncrementFallbackFailures()
		}
	}

	collector.UpdateTotalDuration(totalDuration)
	collector.UpdateRunDuration(update.RunDuration)

}

// 重置metricExchange
func (m *metricExchange) Reset() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// 重置metricCollectors
	for _, collector := range m.metricCollectors {
		collector.Reset()
	}
}

func (m *metricExchange) Requests() *rolling.Number {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.requestsLocked()
}

func (m *metricExchange) requestsLocked() *rolling.Number {
	return m.DefaultCollector().NumRequests()
}

// 错误请求占总请求的比例
func (m *metricExchange) ErrorPercent(now time.Time) int {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var errPct float64
	reqs := m.requestsLocked().Sum(now)
	errs := m.DefaultCollector().Errors().Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs)) * 100
	}

	return int(errPct + 0.5) // 四舍五入
}

// 错误率是否超过限制
func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}
