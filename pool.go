package bytepool

import (
	"expvar"
	"slices"
	"sync/atomic"
)

// BytePool 多档位内存池
type BytePool struct {
	pools          map[int]*Pool[*[]byte]
	stats          map[int]*PoolStats
	sizes          []int
	sizesLen       int
	discardedCount int64 // 丢弃的数量，超过 maxPoolSize 的
	maxPoolSize    int
	recentLengths  *RingQueue[int] // 最近 256 个 get 操作的长度统计
	totalGet       int64           // 总的有效 get 操作数量
	totalPut       int64           // 总的有效 put 操作数量
}

// PoolStats 内存池统计信息
type PoolStats struct {
	Get int64 `json:"get"`
	Put int64 `json:"put"`
}

// SizePowerOfTwo 返回从7开始到21的2的幂次方序列
func SizePowerOfTwo() []int {
	return []int{
		128,     // 2^7
		256,     // 2^8
		512,     // 2^9
		1024,    // 2^10
		2048,    // 2^11
		4096,    // 2^12
		8192,    // 2^13
		16384,   // 2^14
		32768,   // 2^15
		65536,   // 2^16
		131072,  // 2^17
		262144,  // 2^18
		524288,  // 2^19
		1048576, // 2^20
		2097152, // 2^21
	}
}

func SizeStream() []int {
	return []int{
		1024,
		4096,
		16384,
		32768,
		65536,
		131072,
		262144,
	}
}

// NewPools 传入分层的数组
// 超过数组的最大值，就不会放回内存池
func NewPools(sizes []int) *BytePool {
	l := len(sizes)
	if l < 1 {
		panic("sizes is empty")
	}
	pool := BytePool{
		pools:         make(map[int]*Pool[*[]byte]),
		stats:         make(map[int]*PoolStats),
		sizes:         make([]int, l),
		sizesLen:      l,
		recentLengths: NewRingQueue[int](256), // 初始化环形队列，容量为 256
	}

	copy(pool.sizes, sizes)
	slices.Sort(pool.sizes)

	pool.maxPoolSize = pool.sizes[l-1]

	for _, size := range pool.sizes {
		pool.pools[size] = NewPool(func() *[]byte {
			buf := make([]byte, size)
			return &buf
		})
		pool.stats[size] = &PoolStats{}
	}
	return &pool
}

func (p *BytePool) Alloc(size int) *Buffer {
	if size <= 0 || size > p.maxPoolSize {
		return nil
	}
	for _, size := range p.sizes {
		for range size {
			buf := p.Get(size)
			p.Put(buf)
		}
	}
	return nil
}

// findBestSize 根据需要的长度找到最合适的档位
func (p *BytePool) findBestSize(length int) int {
	for _, size := range p.sizes {
		if size >= length {
			return size
		}
	}
	return p.sizes[len(p.sizes)-1]
}

// Get 从池中获取指定长度的 []byte
func (p *BytePool) Get(length int) []byte {
	if length <= 0 {
		return nil
	}

	// 记录请求的长度到环形队列
	p.recentLengths.Push(length)

	if length > p.maxPoolSize {
		atomic.AddInt64(&p.discardedCount, 1)
		return make([]byte, length)
	}

	size := p.findBestSize(length)
	if pool, ok := p.pools[size]; ok {
		// 只有真正从内存池获取时才统计
		atomic.AddInt64(&p.stats[size].Get, 1)
		atomic.AddInt64(&p.totalGet, 1)

		buf := *pool.Get()
		return buf[:length]
	}

	return make([]byte, length)
}

func (p *BytePool) GetBuffer(length int) *Buffer {
	buf := p.Get(length)
	return NewBuffer(buf, p)
}

func (p *BytePool) ReleaseBuffer(buf *Buffer) {
	buf.Release()
}

// Put 将 []byte 放回池中
func (p *BytePool) Put(buf []byte) {
	if buf == nil || cap(buf) == 0 {
		return
	}

	capacity := cap(buf)

	// 超过最大池化大小，直接丢弃
	if capacity > p.maxPoolSize {
		atomic.AddInt64(&p.discardedCount, 1)
		return
	}

	if pool, ok := p.pools[capacity]; ok {
		// 只有真正放回内存池时才统计
		atomic.AddInt64(&p.stats[capacity].Put, 1)
		atomic.AddInt64(&p.totalPut, 1)

		// 重置切片长度为容量，清零内容
		buf = buf[:capacity]
		clear(buf)
		pool.Put(&buf)
	}
	// 如果容量不匹配任何档位，直接丢弃让 GC 回收
}

// GetAvailableSizes 获取所有可用的档位大小
func (p *BytePool) GetAvailableSizes() []int {
	return slices.Clone(p.sizes)
}

// GetDiscardedCount 获取丢弃的数量统计
func (p *BytePool) GetDiscardedCount() int64 {
	return atomic.LoadInt64(&p.discardedCount)
}

// GetPoolStats 获取池的统计信息（用于调试）
func (p *BytePool) GetPoolStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 各档位统计
	poolStats := make(map[int]map[string]int64)
	for size, stat := range p.stats {
		poolStats[size] = map[string]int64{
			"get": atomic.LoadInt64(&stat.Get),
			"put": atomic.LoadInt64(&stat.Put),
		}
	}
	stats["pools"] = poolStats
	stats["discarded"] = atomic.LoadInt64(&p.discardedCount)

	// 添加总统计
	totalGet := atomic.LoadInt64(&p.totalGet)
	totalPut := atomic.LoadInt64(&p.totalPut)
	stats["total_get"] = totalGet
	stats["total_put"] = totalPut

	// 添加最近 256 个 get 操作的长度统计
	recentLengths := p.recentLengths.Bytes()
	stats["recent_lengths"] = recentLengths

	return stats
}

func (p *BytePool) Expvar(prefix string) *BytePool {
	expvar.Publish(prefix+"pool_stats", expvar.Func(func() any {
		return p.GetPoolStats()
	}))
	return p
}
