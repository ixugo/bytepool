package bytepool

import (
	"sync/atomic"
)

// RingQueue 简化的无锁环形队列
// 只使用递增的写位置，允许脏读
type RingQueue[T any] struct {
	data     []T
	size     int64
	writePos int64 // 递增的写入位置，永不回退
}

// NewRingQueue 创建一个新的环形队列
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

// Push 向队列尾部添加元素
func (rq *RingQueue[T]) Push(item T) {
	// 获取当前写入位置并递增
	pos := atomic.AddInt64(&rq.writePos, 1) - 1

	// 写入数据到实际位置（取模）
	actualPos := pos % rq.size
	rq.data[actualPos] = item
}

// Bytes 获取当前队列的所有数据（允许脏读）
func (rq *RingQueue[T]) Bytes() []T {
	currentWritePos := atomic.LoadInt64(&rq.writePos)

	// 如果还没有写入任何数据
	if currentWritePos == 0 {
		return nil
	}

	// 判断是否产生了循环
	if currentWritePos <= rq.size {
		// 没有循环，直接返回从头到当前写入位置的数据
		result := make([]T, currentWritePos)
		copy(result, rq.data[:currentWritePos])
		return result
	}

	// 产生了循环，队列已满
	// writePos 是下一个要写入的位置，也就是最老数据的位置（第一个值）
	result := make([]T, rq.size)
	firstPos := currentWritePos % rq.size // 第一个值的位置

	// 从第一个值开始，按顺序读取 size 个元素
	for i := int64(0); i < rq.size; i++ {
		actualPos := (firstPos + i) % rq.size
		result[i] = rq.data[actualPos]
	}

	return result
}

// Len 返回当前元素数量
func (rq *RingQueue[T]) Len() int {
	writePos := atomic.LoadInt64(&rq.writePos)
	if writePos <= rq.size {
		return int(writePos)
	}
	return int(rq.size)
}

// Cap 返回队列容量
func (rq *RingQueue[T]) Cap() int {
	return int(rq.size)
}

// IsFull 检查队列是否已满
func (rq *RingQueue[T]) IsFull() bool {
	return atomic.LoadInt64(&rq.writePos) >= rq.size
}

// IsEmpty 检查队列是否为空
func (rq *RingQueue[T]) IsEmpty() bool {
	return atomic.LoadInt64(&rq.writePos) == 0
}

// Clear 清空队列
func (rq *RingQueue[T]) Clear() {
	atomic.StoreInt64(&rq.writePos, 0)
	clear(rq.data)
}
