package bytepool

import (
	"sync/atomic"
)

// RingQueue is a simplified lock-free ring queue
// Only uses incrementing write position, allows dirty reads
type RingQueue[T any] struct {
	data     []T
	size     int64
	writePos int64 // incrementing write position, never rolls back
}

// NewRingQueue creates a new ring queue with the specified size
func NewRingQueue[T any](size int) *RingQueue[T] {
	if size <= 0 {
		panic("ring queue size must be positive")
	}
	return &RingQueue[T]{
		data:     make([]T, size),
		size:     int64(size),
		writePos: 0,
	}
}

// Push adds an element to the tail of the queue
func (rq *RingQueue[T]) Push(item T) {
	// get current write position and increment
	pos := atomic.AddInt64(&rq.writePos, 1) - 1

	// write data to actual position (modulo)
	actualPos := pos % rq.size
	rq.data[actualPos] = item
}

// Bytes returns all current data in the queue (allows dirty reads)
func (rq *RingQueue[T]) Bytes() []T {
	currentWritePos := atomic.LoadInt64(&rq.writePos)

	// if no data has been written yet
	if currentWritePos == 0 {
		return nil
	}

	// check if wrapping has occurred
	if currentWritePos <= rq.size {
		// no wrapping, return data from start to current write position
		result := make([]T, currentWritePos)
		copy(result, rq.data[:currentWritePos])
		return result
	}

	// wrapping has occurred, queue is full
	// writePos is the next position to write, also the position of oldest data (first value)
	result := make([]T, rq.size)
	firstPos := currentWritePos % rq.size // position of first value

	// read size elements in order starting from first value
	for i := int64(0); i < rq.size; i++ {
		actualPos := (firstPos + i) % rq.size
		result[i] = rq.data[actualPos]
	}

	return result
}

// Len returns the current number of elements
func (rq *RingQueue[T]) Len() int {
	writePos := atomic.LoadInt64(&rq.writePos)
	if writePos <= rq.size {
		return int(writePos)
	}
	return int(rq.size)
}

// Cap returns the queue capacity
func (rq *RingQueue[T]) Cap() int {
	return int(rq.size)
}

// IsFull checks if the queue is full
func (rq *RingQueue[T]) IsFull() bool {
	return atomic.LoadInt64(&rq.writePos) >= rq.size
}

// IsEmpty checks if the queue is empty
func (rq *RingQueue[T]) IsEmpty() bool {
	return atomic.LoadInt64(&rq.writePos) == 0
}

// Clear empties the queue
func (rq *RingQueue[T]) Clear() {
	atomic.StoreInt64(&rq.writePos, 0)
	clear(rq.data)
}
