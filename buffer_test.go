package bytepool

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// BenchmarkBufferCreation 测试 Buffer 创建性能
func BenchmarkBufferCreation(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	data := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := NewBuffer(data, pool)
		buf.Release()
	}
}

// BenchmarkBufferBytes 测试 Buffer.Bytes() 性能
func BenchmarkBufferBytes(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	data := make([]byte, 1024)
	buf := NewBuffer(data, pool)
	defer buf.Release()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		bytes, release := buf.Bytes()
		_ = bytes
		release()
	}
}

// BenchmarkBytePoolGet 测试 BytePool.Get() 性能
func BenchmarkBytePoolGet(b *testing.B) {
	sizes := []int{128, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := NewPools(SizePowerOfTwo())

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.Get(size)
				pool.Put(buf)
			}
		})
	}
}

// BenchmarkBytePoolGetBuffer 测试 BytePool.GetBuffer() 性能
func BenchmarkBytePoolGetBuffer(b *testing.B) {
	sizes := []int{128, 1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := NewPools(SizePowerOfTwo())

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(size)
				buf.Release()
			}
		})
	}
}

// BenchmarkBytePoolVsDirectAlloc 对比池化分配与直接分配的性能
func BenchmarkBytePoolVsDirectAlloc(b *testing.B) {
	size := 4096

	b.Run("pool", func(b *testing.B) {
		pool := NewPools(SizePowerOfTwo())

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := pool.Get(size)
			pool.Put(buf)
		}
	})

	b.Run("direct", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, size)
			_ = buf
		}
	})
}

// BenchmarkConcurrentBytePool 测试并发场景下的性能
func BenchmarkConcurrentBytePool(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	size := 4096

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(size)
			pool.Put(buf)
		}
	})
}

// BenchmarkConcurrentBuffer 测试并发 Buffer 操作
func BenchmarkConcurrentBuffer(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	data := make([]byte, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := NewBuffer(data, pool)
			bytes, release := buf.Bytes()
			_ = bytes
			release()
			buf.Release()
		}
	})
}

// BenchmarkBufferReuse 测试 Buffer 重用性能
func BenchmarkBufferReuse(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 模拟实际使用场景：获取、写入、读取、释放
		buf := pool.GetBuffer(1024)

		// 模拟写入数据
		data, release := buf.Bytes()
		copy(data, "test data")
		release()

		buf.Release()
	}
}

// BenchmarkMemoryUsage 测试内存使用情况
func BenchmarkMemoryUsage(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := pool.GetBuffer(4096)
		buf.Release()
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// BenchmarkDifferentSizes 测试不同大小的性能差异
func BenchmarkDifferentSizes(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	sizes := []int{128, 512, 2048, 8192, 32768, 131072}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(size)
				buf.Release()
			}
		})
	}
}

// BenchmarkHighContention 测试高竞争场景
func BenchmarkHighContention(b *testing.B) {
	pool := NewPools(SizePowerOfTwo())
	size := 4096
	numGoroutines := runtime.NumCPU() * 4

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/numGoroutines; j++ {
				buf := pool.GetBuffer(size)
				buf.Release()
			}
		}()
	}
	wg.Wait()
}
