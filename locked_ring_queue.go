package bytepool

import (
	"sync"
)

var _ RingQueuer = (*LockedRingQueue[int])(nil)

// LockedRingQueue is a thread-safe ring queue using mutex locks.
// It provides strong consistency guarantees at the cost of some performance
// compared to the lock-free version.
type LockedRingQueue[T any] struct {
	data     []T
	size     int
	writePos int // current write position
	readPos  int // current read position (oldest data)
	count    int // current number of elements
	mu       sync.RWMutex
}

// NewLockedRingQueue creates a new locked ring queue with the specified size
func NewLockedRingQueue[T any](size int) *LockedRingQueue[T] {
	if size <= 0 {
		panic("ring queue size must be positive")
	}
	return &LockedRingQueue[T]{
		data:     make([]T, size),
		size:     size,
		writePos: 0,
		readPos:  0,
		count:    0,
	}
}

// Push adds an element to the tail of the queue
// If the queue is full, it overwrites the oldest element
func (lrq *LockedRingQueue[T]) Push(item T) {
	lrq.mu.Lock()
	defer lrq.mu.Unlock()

	lrq.data[lrq.writePos] = item
	lrq.writePos = (lrq.writePos + 1) % lrq.size

	if lrq.count < lrq.size {
		lrq.count++
	} else {
		// queue is full, move read position to maintain ring behavior
		lrq.readPos = (lrq.readPos + 1) % lrq.size
	}
}

// Pop removes and returns the oldest element from the queue
// Returns zero value and false if queue is empty
func (lrq *LockedRingQueue[T]) Pop() (T, bool) {
	lrq.mu.Lock()
	defer lrq.mu.Unlock()

	var zero T
	if lrq.count == 0 {
		return zero, false
	}

	item := lrq.data[lrq.readPos]
	lrq.data[lrq.readPos] = zero // clear the slot
	lrq.readPos = (lrq.readPos + 1) % lrq.size
	lrq.count--

	return item, true
}

// Peek returns the oldest element without removing it
// Returns zero value and false if queue is empty
func (lrq *LockedRingQueue[T]) Peek() (T, bool) {
	lrq.mu.RLock()
	defer lrq.mu.RUnlock()

	var zero T
	if lrq.count == 0 {
		return zero, false
	}

	return lrq.data[lrq.readPos], true
}

// Bytes returns all current data in the queue in order (oldest to newest)
func (lrq *LockedRingQueue[T]) Bytes() []T {
	lrq.mu.RLock()
	defer lrq.mu.RUnlock()

	if lrq.count == 0 {
		return nil
	}

	result := make([]T, lrq.count)
	for i := 0; i < lrq.count; i++ {
		pos := (lrq.readPos + i) % lrq.size
		result[i] = lrq.data[pos]
	}

	return result
}

// Len returns the current number of elements
func (lrq *LockedRingQueue[T]) Len() int {
	lrq.mu.RLock()
	defer lrq.mu.RUnlock()
	return lrq.count
}

// Cap returns the queue capacity
func (lrq *LockedRingQueue[T]) Cap() int {
	return lrq.size
}

// IsFull checks if the queue is full
func (lrq *LockedRingQueue[T]) IsFull() bool {
	lrq.mu.RLock()
	defer lrq.mu.RUnlock()
	return lrq.count == lrq.size
}

// IsEmpty checks if the queue is empty
func (lrq *LockedRingQueue[T]) IsEmpty() bool {
	lrq.mu.RLock()
	defer lrq.mu.RUnlock()
	return lrq.count == 0
}

// Clear empties the queue
func (lrq *LockedRingQueue[T]) Clear() {
	lrq.mu.Lock()
	defer lrq.mu.Unlock()

	var zero T
	for i := range lrq.data {
		lrq.data[i] = zero
	}
	lrq.writePos = 0
	lrq.readPos = 0
	lrq.count = 0
}
