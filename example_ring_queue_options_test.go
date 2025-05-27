package bytepool

import (
	"fmt"
	"testing"
)

// ExampleWithRingQueueType demonstrates how to configure different ring queue types
func ExampleWithRingQueueType() {
	// Create BytePool with default lock-free ring queue
	defaultPool := NewPools([]int{128, 256, 512})

	// Create BytePool with mutex-based ring queue for strong consistency
	mutexPool := NewPools([]int{128, 256, 512}, WithRingQueueType(MutexRingQueue))

	// Use both pools
	buf1 := defaultPool.Get(100)
	buf2 := mutexPool.Get(100)

	defaultPool.Put(buf1)
	mutexPool.Put(buf2)

	// Get statistics
	stats1 := defaultPool.GetPoolStats()
	stats2 := mutexPool.GetPoolStats()

	fmt.Printf("Default pool recent lengths: %v\n", len(stats1["recent_lengths"].([]int)))
	fmt.Printf("Mutex pool recent lengths: %v\n", len(stats2["recent_lengths"].([]int)))

	// Output:
	// Default pool recent lengths: 1
	// Mutex pool recent lengths: 1
}

// ExampleWithRingQueue demonstrates how to use a custom ring queue
func ExampleWithRingQueue() {
	// Create a custom locked ring queue with specific capacity
	customQueue := NewLockedRingQueue[int](64)

	// Create BytePool with custom ring queue
	pool := NewPools([]int{128, 256, 512}, WithRingQueue(customQueue))

	// Use the pool
	buf := pool.Get(150)
	pool.Put(buf)

	// Access the custom queue directly
	fmt.Printf("Custom queue capacity: %d\n", customQueue.Cap())
	fmt.Printf("Custom queue length: %d\n", customQueue.Len())

	// Output:
	// Custom queue capacity: 64
	// Custom queue length: 1
}

// Example_ringQueueComparison demonstrates the differences between lock-free and mutex-based queues
func Example_ringQueueComparison() {
	// Lock-free ring queue
	lockFreeQueue := NewRingQueue[string](3)
	lockFreeQueue.Push("a")
	lockFreeQueue.Push("b")
	lockFreeQueue.Push("c")

	fmt.Printf("Lock-free queue data: %v\n", lockFreeQueue.Bytes())
	fmt.Printf("Lock-free queue length: %d\n", lockFreeQueue.Len())

	// Mutex-based ring queue with additional operations
	mutexQueue := NewLockedRingQueue[string](3)
	mutexQueue.Push("x")
	mutexQueue.Push("y")
	mutexQueue.Push("z")

	fmt.Printf("Mutex queue data: %v\n", mutexQueue.Bytes())

	// Additional operations only available in mutex-based queue
	if item, ok := mutexQueue.Peek(); ok {
		fmt.Printf("Oldest item (peek): %s\n", item)
	}

	if item, ok := mutexQueue.Pop(); ok {
		fmt.Printf("Popped item: %s\n", item)
		fmt.Printf("After pop, length: %d\n", mutexQueue.Len())
	}

	// Output:
	// Lock-free queue data: [a b c]
	// Lock-free queue length: 3
	// Mutex queue data: [x y z]
	// Oldest item (peek): x
	// Popped item: x
	// After pop, length: 2
}

func TestRingQueueOptions(t *testing.T) {
	// Test that different configurations work correctly

	// Test lock-free configuration
	lockFreePool := NewPools([]int{128, 256}, WithRingQueueType(LockFreeRingQueue))
	buf := lockFreePool.Get(100)
	lockFreePool.Put(buf)

	stats := lockFreePool.GetPoolStats()
	if len(stats["recent_lengths"].([]int)) != 1 {
		t.Errorf("Expected 1 recent length, got %d", len(stats["recent_lengths"].([]int)))
	}

	// Test mutex configuration
	mutexPool := NewPools([]int{128, 256}, WithRingQueueType(MutexRingQueue))
	buf = mutexPool.Get(200)
	mutexPool.Put(buf)

	stats = mutexPool.GetPoolStats()
	if len(stats["recent_lengths"].([]int)) != 1 {
		t.Errorf("Expected 1 recent length, got %d", len(stats["recent_lengths"].([]int)))
	}

	// Test custom ring queue
	customQueue := NewLockedRingQueue[int](10)
	customPool := NewPools([]int{128, 256}, WithRingQueue(customQueue))
	buf = customPool.Get(150)
	customPool.Put(buf)

	if customQueue.Len() != 1 {
		t.Errorf("Expected custom queue length 1, got %d", customQueue.Len())
	}
}
