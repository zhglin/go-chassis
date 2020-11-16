package metricCollector

import (
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/v2/third_party/forked/afex/hystrix-go/hystrix/rolling"
)

// DefaultMetricCollector holds information about the circuit state.
// This implementation of MetricCollector is the canonical source of information about the circuit.
// It is used for for all internal hystrix operations
// including circuit health checks and metrics sent to the hystrix dashboard.
//
// Metric Collectors do not need Mutexes as they are updated by circuits within a locked context.
// 默认的计数器
type DefaultMetricCollector struct {
	mutex       *sync.RWMutex
	name        string
	numRequests *rolling.Number // 总的请求数
	errors      *rolling.Number // 异常的请求数

	successes     *rolling.Number // eun成功的请求数
	failures      *rolling.Number // run失败的请求数
	rejects       *rolling.Number // 超过并发限制的请求数
	shortCircuits *rolling.Number // 被熔断的请求数
	timeouts      *rolling.Number // 超时的请求数

	fallbackSuccesses *rolling.Number // fallback执行成功的请求数
	fallbackFailures  *rolling.Number // fallback执行失败的请求数
	totalDuration     *rolling.Timing // 请求总的花费时间
	runDuration       *rolling.Timing // run函数的执行时间
}

// 创建metricCollector
func newDefaultMetricCollector(name string) MetricCollector {
	m := &DefaultMetricCollector{}
	m.mutex = &sync.RWMutex{}
	m.Reset()
	m.name = name
	return m
}

// NumRequests returns the rolling number of requests
func (d *DefaultMetricCollector) NumRequests() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.numRequests
}

// Errors returns the rolling number of errors
func (d *DefaultMetricCollector) Errors() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.errors
}

// Successes returns the rolling number of successes
func (d *DefaultMetricCollector) Successes() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.successes
}

// Failures returns the rolling number of failures
func (d *DefaultMetricCollector) Failures() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.failures
}

// Rejects returns the rolling number of rejects
func (d *DefaultMetricCollector) Rejects() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.rejects
}

// ShortCircuits returns the rolling number of short circuits
func (d *DefaultMetricCollector) ShortCircuits() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.shortCircuits
}

// Timeouts returns the rolling number of timeouts
func (d *DefaultMetricCollector) Timeouts() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.timeouts
}

// FallbackSuccesses returns the rolling number of fallback successes
func (d *DefaultMetricCollector) FallbackSuccesses() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.fallbackSuccesses
}

// FallbackFailures returns the rolling number of fallback failures
func (d *DefaultMetricCollector) FallbackFailures() *rolling.Number {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.fallbackFailures
}

// TotalDuration returns the rolling total duration
func (d *DefaultMetricCollector) TotalDuration() *rolling.Timing {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.totalDuration
}

// RunDuration returns the rolling run duration
func (d *DefaultMetricCollector) RunDuration() *rolling.Timing {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.runDuration
}

// IncrementAttempts increments the number of requests seen in the latest time bucket.
func (d *DefaultMetricCollector) IncrementAttempts() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.numRequests.Increment(1)
}

// IncrementErrors increments the number of errors seen in the latest time bucket.
// Errors are any result from an attempt that is not a success.
// 异常的请求数
func (d *DefaultMetricCollector) IncrementErrors() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.errors.Increment(1)
}

// IncrementSuccesses increments the number of successes seen in the latest time bucket.
// 增加请求成功的计数
func (d *DefaultMetricCollector) IncrementSuccesses() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.successes.Increment(1)
}

// IncrementFailures increments the number of failures seen in the latest time bucket.
// run执行失败的计数
func (d *DefaultMetricCollector) IncrementFailures() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.failures.Increment(1)
}

// IncrementRejects increments the number of rejected requests seen in the latest time bucket.
// 增加超过并发限制的计数
func (d *DefaultMetricCollector) IncrementRejects() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.rejects.Increment(1)
}

// IncrementShortCircuits increments the number of rejected requests seen in the latest time bucket.
// 增加被熔断的计数
func (d *DefaultMetricCollector) IncrementShortCircuits() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.shortCircuits.Increment(1)
}

// IncrementTimeouts increments the number of requests that timed out in the latest time bucket.
// 增加超时的计数
func (d *DefaultMetricCollector) IncrementTimeouts() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.timeouts.Increment(1)
}

// IncrementFallbackSuccesses increments the number of successful calls to the fallback function in the latest time bucket.
// 增加fallback执行成功的计数
func (d *DefaultMetricCollector) IncrementFallbackSuccesses() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.fallbackSuccesses.Increment(1)
}

// IncrementFallbackFailures increments the number of failed calls to the fallback function in the latest time bucket.
func (d *DefaultMetricCollector) IncrementFallbackFailures() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.fallbackFailures.Increment(1)
}

// UpdateTotalDuration updates the total amount of time this circuit has been running.
// 记录每个请求的总的花费时间
func (d *DefaultMetricCollector) UpdateTotalDuration(timeSinceStart time.Duration) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.totalDuration.Add(timeSinceStart)
}

// UpdateRunDuration updates the amount of time the latest request took to complete.
// 每个请求run函数的执行时间
func (d *DefaultMetricCollector) UpdateRunDuration(runDuration time.Duration) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	d.runDuration.Add(runDuration)
}

// Reset resets all metrics in this collector to 0.
// 重置metricCollector
func (d *DefaultMetricCollector) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.numRequests = rolling.NewNumber()
	d.errors = rolling.NewNumber()
	d.successes = rolling.NewNumber()
	d.rejects = rolling.NewNumber()
	d.shortCircuits = rolling.NewNumber()
	d.failures = rolling.NewNumber()
	d.timeouts = rolling.NewNumber()
	d.fallbackSuccesses = rolling.NewNumber()
	d.fallbackFailures = rolling.NewNumber()
	d.totalDuration = rolling.NewTiming()
	d.runDuration = rolling.NewTiming()
}
