package rolling

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Timing maintains time Durations for each time bucket.
// The Durations are kept in an array to allow for a variety of
// statistics to be calculated from the source data.
type Timing struct {
	Buckets map[int64]*timingBucket // key=>时间戳
	Mutex   *sync.RWMutex

	CachedSortedDurations []time.Duration // 排序后的缓存 1s过期
	LastCachedTime        int64           // cache设置时间
}

type timingBucket struct {
	Durations []time.Duration
}

// NewTiming creates a RollingTiming struct.
func NewTiming() *Timing {
	r := &Timing{
		Buckets: make(map[int64]*timingBucket),
		Mutex:   &sync.RWMutex{},
	}
	return r
}

type byDuration []time.Duration

func (c byDuration) Len() int           { return len(c) }
func (c byDuration) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c byDuration) Less(i, j int) bool { return c[i] < c[j] }

// SortedDurations returns an array of time.Duration sorted from shortest
// to longest that have occurred in the last 60 seconds.
// 对60s内的花费时长进行排序
func (r *Timing) SortedDurations() []time.Duration {
	r.Mutex.RLock()
	t := r.LastCachedTime
	r.Mutex.RUnlock()

	// 距离上次排序未超过1s 直接使用cache
	if t+time.Duration(1*time.Second).Nanoseconds() > time.Now().UnixNano() {
		// don't recalculate if current cache is still fresh
		return r.CachedSortedDurations
	}

	var durations byDuration
	now := time.Now()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for timestamp, b := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp >= now.Unix()-60 {
			for _, d := range b.Durations {
				durations = append(durations, d)
			}
		}
	}

	sort.Sort(durations)

	// 缓存排序结果
	r.CachedSortedDurations = durations
	r.LastCachedTime = time.Now().UnixNano()

	return r.CachedSortedDurations
}

// 当前时间对应的bucket
func (r *Timing) getCurrentBucket() *timingBucket {
	r.Mutex.RLock()
	now := time.Now()
	bucket, exists := r.Buckets[now.Unix()]
	r.Mutex.RUnlock()

	if !exists {
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		r.Buckets[now.Unix()] = &timingBucket{}
		bucket = r.Buckets[now.Unix()]
	}

	return bucket
}

// 删除60s之前的buckets
func (r *Timing) removeOldBuckets() {
	now := time.Now()

	for timestamp := range r.Buckets {
		// TODO: configurable rolling window
		if timestamp <= now.Unix()-60 {
			delete(r.Buckets, timestamp)
		}
	}
}

// Add appends the time.Duration given to the current time bucket.
// 在bucket中增加一个耗时
func (r *Timing) Add(duration time.Duration) {
	b := r.getCurrentBucket()

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	b.Durations = append(b.Durations, duration)
	r.removeOldBuckets() // 清理过期buckets
}

// Percentile computes the percentile given with a linear interpolation.
//it returns million seconds
// 百分位p下的耗时
func (r *Timing) Percentile(p float64) uint32 {
	sortedDurations := r.SortedDurations()
	length := len(sortedDurations)
	if length <= 0 {
		return 0
	}

	pos := r.ordinal(len(sortedDurations), p) - 1
	return uint32(sortedDurations[pos].Nanoseconds() / 1000000)
}

func (r *Timing) ordinal(length int, percentile float64) int64 {
	if percentile == 0 && length > 0 {
		return 1
	}

	return int64(math.Ceil((percentile / float64(100)) * float64(length)))
}

// Mean computes the average timing in the last 60 seconds.
// 60s内的平均响应时间
func (r *Timing) Mean() uint32 {
	sortedDurations := r.SortedDurations()
	var sum time.Duration
	for _, d := range sortedDurations {
		sum += d
	}

	length := int64(len(sortedDurations))
	if length == 0 {
		return 0
	}

	return uint32(sum.Nanoseconds()/length) / 1000000
}
