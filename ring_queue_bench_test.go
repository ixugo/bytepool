package bytepool

import (
	"sync"
	"testing"
)

// Single-threaded benchmarks
func BenchmarkRingQueue_Push_LockFree(b *testing.B) {
	queue := NewRingQueue[int](1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queue.Push(i)
	}
}

func BenchmarkRingQueue_Push_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queue.Push(i)
	}
}

func BenchmarkRingQueue_Bytes_LockFree(b *testing.B) {
	queue := NewRingQueue[int](1000)
	// Pre-fill the queue
	for i := 0; i < 1000; i++ {
		queue.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.Bytes()
	}
}

func BenchmarkRingQueue_Bytes_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](1000)
	// Pre-fill the queue
	for i := 0; i < 1000; i++ {
		queue.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.Bytes()
	}
}

func BenchmarkRingQueue_Pop_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](1000)
	// Pre-fill the queue
	for i := 0; i < 1000; i++ {
		queue.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := queue.Pop(); !ok {
			// Refill if empty
			for j := 0; j < 1000; j++ {
				queue.Push(j)
			}
		}
	}
}

// Concurrent benchmarks
func BenchmarkRingQueue_ConcurrentPush_LockFree(b *testing.B) {
	queue := NewRingQueue[int](10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			queue.Push(i)
			i++
		}
	})
}

func BenchmarkRingQueue_ConcurrentPush_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			queue.Push(i)
			i++
		}
	})
}

func BenchmarkRingQueue_ConcurrentBytes_LockFree(b *testing.B) {
	queue := NewRingQueue[int](1000)
	// Pre-fill the queue
	for i := 0; i < 1000; i++ {
		queue.Push(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = queue.Bytes()
		}
	})
}

func BenchmarkRingQueue_ConcurrentBytes_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](1000)
	// Pre-fill the queue
	for i := 0; i < 1000; i++ {
		queue.Push(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = queue.Bytes()
		}
	})
}

// Mixed operations benchmarks
func BenchmarkRingQueue_MixedOperations_Locked(b *testing.B) {
	queue := NewLockedRingQueue[int](1000)
	var wg sync.WaitGroup

	b.ResetTimer()

	// Start background pushers
	numPushers := 2
	wg.Add(numPushers)
	for i := 0; i < numPushers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/numPushers; j++ {
				queue.Push(j)
			}
		}()
	}

	// Start background poppers
	numPoppers := 2
	wg.Add(numPoppers)
	for i := 0; i < numPoppers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/numPoppers; j++ {
				queue.Pop()
			}
		}()
	}

	wg.Wait()
}

// BytePool benchmarks with different ring queue types
func BenchmarkBytePool_LockFreeRingQueue(b *testing.B) {
	pool := NewPools([]int{128, 512, 1024, 4096}, WithRingQueueType(LockFreeRingQueue))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(256)
			pool.Put(buf)
		}
	})
}

func BenchmarkBytePool_MutexRingQueue(b *testing.B) {
	pool := NewPools([]int{128, 512, 1024, 4096}, WithRingQueueType(MutexRingQueue))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(256)
			pool.Put(buf)
		}
	})
}

// Memory allocation comparison
func BenchmarkRingQueue_MemoryAllocation_LockFree(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		queue := NewRingQueue[int](100)
		for j := 0; j < 50; j++ {
			queue.Push(j)
		}
		_ = queue.Bytes()
	}
}

func BenchmarkRingQueue_MemoryAllocation_Locked(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		queue := NewLockedRingQueue[int](100)
		for j := 0; j < 50; j++ {
			queue.Push(j)
		}
		_ = queue.Bytes()
	}
}
