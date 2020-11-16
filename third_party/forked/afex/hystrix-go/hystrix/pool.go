package hystrix

// 并发限制
type executorPool struct {
	Name    string // 唯一标识
	Metrics *poolMetrics
	Max     int            // tickets最大数量
	Tickets chan *struct{} // 执行前获取ticket
}

const ConcurrentRequestsLimit = 5000

// 创建个executorPool
func newExecutorPool(name string) *executorPool {
	p := &executorPool{}
	p.Name = name
	p.Metrics = newPoolMetrics(name)
	p.Max = getSettings(name).MaxConcurrentRequests
	if p.Max > ConcurrentRequestsLimit {
		p.Max = ConcurrentRequestsLimit
	}

	// 提前写入max个tickets
	p.Tickets = make(chan *struct{}, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}

	return p
}

// 执行完归还ticket
func (p *executorPool) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Metrics.Updates <- poolMetricsUpdate{
		activeCount: p.ActiveCount(),
	}
	p.Tickets <- ticket // 归还ticket
}

// 当前剩余的tickets
func (p *executorPool) ActiveCount() int {
	return p.Max - len(p.Tickets)
}
