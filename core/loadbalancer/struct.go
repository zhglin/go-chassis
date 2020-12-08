package loadbalancer

import (
	"time"
)

// ProtocolStats store protocol stats
// 每个请求的耗时信息
type ProtocolStats struct {
	Latency    []time.Duration // 每次的耗时 只保留最近10条
	Addr       string          // 请求地址
	AvgLatency time.Duration   // 平均耗时
}

// CalculateAverageLatency make avg latency
// 计算平均耗时
func (ps *ProtocolStats) CalculateAverageLatency() {
	var sum time.Duration
	for i := 0; i < len(ps.Latency); i++ {
		sum = sum + ps.Latency[i]
	}
	if len(ps.Latency) == 0 {
		return
	}
	ps.AvgLatency = time.Duration(sum.Nanoseconds() / int64(len(ps.Latency)))
}

// SaveLatency save latest 10 record
// 添加本次的请求的耗时
func (ps *ProtocolStats) SaveLatency(l time.Duration) {
	if len(ps.Latency) >= 10 {
		//save latest 10 latencies
		ps.Latency = ps.Latency[1:]
	}
	ps.Latency = append(ps.Latency, l)
}
