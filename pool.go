package bytepool

import (
	"expvar"
	"slices"
	"sync/atomic"
)

type RingQueuer interface {
	Push(item int)
	Bytes() []int
}

// BytePool is a multi-tier memory pool
type BytePool struct {
	pools          map[int]*Pool[*[]byte]
	stats          map[int]*PoolStats
	sizes          []int
	sizesLen       int
	discardedCount int64 // count of discarded items that exceed maxPoolSize
	maxPoolSize    int
	recentLengths  RingQueuer // statistics of recent 256 get operation lengths
	totalGet       int64      // total number of valid get operations
	totalPut       int64      // total number of valid put operations
}

// PoolStats represents memory pool statistics
type PoolStats struct {
	Get int64 `json:"get"`
	Put int64 `json:"put"`
}

type Option func(*BytePool)

// RingQueueType represents the type of ring queue to use
type RingQueueType int

const (
	// LockFreeRingQueue uses atomic operations for maximum performance (default)
	LockFreeRingQueue RingQueueType = iota
	// MutexRingQueue uses mutex locks for strong consistency
	MutexRingQueue
)

func WithRingQueue(ringQueuer RingQueuer) Option {
	return func(p *BytePool) {
		p.recentLengths = ringQueuer
	}
}

// WithRingQueueType sets the type of ring queue to use
func WithRingQueueType(queueType RingQueueType) Option {
	return func(p *BytePool) {
		switch queueType {
		case LockFreeRingQueue:
			p.recentLengths = NewRingQueue[int](256)
		case MutexRingQueue:
			p.recentLengths = NewLockedRingQueue[int](256)
		default:
			p.recentLengths = NewRingQueue[int](256) // default to lock-free
		}
	}
}

// NewPools creates a new BytePool with the given tier sizes
// Items exceeding the maximum size will not be returned to the pool
func NewPools(sizes []int, opts ...Option) *BytePool {
	l := len(sizes)
	if l < 1 {
		panic("sizes is empty")
	}
	pool := BytePool{
		pools:         make(map[int]*Pool[*[]byte]),
		stats:         make(map[int]*PoolStats),
		sizes:         make([]int, l),
		sizesLen:      l,
		recentLengths: NewRingQueue[int](256), // initialize ring queue with capacity 256
	}
	for _, opt := range opts {
		opt(&pool)
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

// Alloc allocates a buffer of the specified size (placeholder implementation)
func (p *BytePool) Alloc(size int) *BytePool {
	if size <= 0 || size > p.maxPoolSize {
		return nil
	}
	for _, size := range p.sizes {
		for range size {
			buf := p.Get(size)
			p.Put(buf)
		}
	}
	return p
}

// findBestSize finds the most suitable tier based on the required length
func (p *BytePool) findBestSize(length int) int {
	for _, size := range p.sizes {
		if size >= length {
			return size
		}
	}
	return p.sizes[len(p.sizes)-1]
}

// Get retrieves a []byte of the specified length from the pool
func (p *BytePool) Get(length int) []byte {
	if length <= 0 {
		return nil
	}

	// record the requested length to the ring queue
	p.recentLengths.Push(length)

	if length > p.maxPoolSize {
		atomic.AddInt64(&p.discardedCount, 1)
		return make([]byte, length)
	}

	size := p.findBestSize(length)
	if pool, ok := p.pools[size]; ok {
		// only count when actually getting from the memory pool
		atomic.AddInt64(&p.stats[size].Get, 1)
		atomic.AddInt64(&p.totalGet, 1)

		buf := *pool.Get()
		return buf[:length]
	}

	return make([]byte, length)
}

// GetBuffer retrieves a Buffer of the specified length from the pool
func (p *BytePool) GetBuffer(length int) *Buffer {
	buf := p.Get(length)
	return NewBuffer(buf, p)
}

// ReleaseBuffer releases a Buffer back to the pool
func (p *BytePool) ReleaseBuffer(buf *Buffer) {
	buf.Release()
}

// Put returns a []byte to the pool
func (p *BytePool) Put(buf []byte) {
	if buf == nil || cap(buf) == 0 {
		return
	}

	capacity := cap(buf)

	// discard if exceeding maximum pool size
	if capacity > p.maxPoolSize {
		atomic.AddInt64(&p.discardedCount, 1)
		return
	}

	if pool, ok := p.pools[capacity]; ok {
		// only count when actually returning to the memory pool
		atomic.AddInt64(&p.stats[capacity].Put, 1)
		atomic.AddInt64(&p.totalPut, 1)

		// reset slice length to capacity and clear content
		buf = buf[:capacity]
		pool.Put(&buf)
	}
	// if capacity doesn't match any tier, discard and let GC collect
}

// GetAvailableSizes returns all available tier sizes
func (p *BytePool) GetAvailableSizes() []int {
	return slices.Clone(p.sizes)
}

// GetDiscardedCount returns the count of discarded items
func (p *BytePool) GetDiscardedCount() int64 {
	return atomic.LoadInt64(&p.discardedCount)
}

// GetPoolStats returns pool statistics (for debugging)
func (p *BytePool) GetPoolStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// statistics for each tier
	poolStats := make(map[int]map[string]int64)
	for size, stat := range p.stats {
		poolStats[size] = map[string]int64{
			"get": atomic.LoadInt64(&stat.Get),
			"put": atomic.LoadInt64(&stat.Put),
		}
	}
	stats["pools"] = poolStats
	stats["discarded"] = atomic.LoadInt64(&p.discardedCount)

	// add total statistics
	totalGet := atomic.LoadInt64(&p.totalGet)
	totalPut := atomic.LoadInt64(&p.totalPut)
	stats["total_get"] = totalGet
	stats["total_put"] = totalPut

	// add statistics of recent 256 get operation lengths
	recentLengths := p.recentLengths.Bytes()
	stats["recent_lengths"] = recentLengths

	return stats
}

// Expvar publishes pool statistics to expvar with the given prefix
func (p *BytePool) Expvar(prefix string) *BytePool {
	expvar.Publish(prefix+"pool_stats", expvar.Func(func() any {
		return p.GetPoolStats()
	}))
	return p
}
