package bytepool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRingQueueBasic(t *testing.T) {
	rq := NewRingQueue[int](3)

	// 测试基本操作
	if !rq.IsEmpty() {
		t.Error("Expected queue to be empty")
	}

	rq.Push(1)
	rq.Push(2)
	rq.Push(3)

	if !rq.IsFull() {
		t.Error("Expected queue to be full")
	}

	data := rq.Bytes()
	expected := []int{1, 2, 3}
	if len(data) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(data))
	}
	for i, v := range expected {
		if data[i] != v {
			t.Errorf("Expected %d at index %d, got %d", v, i, data[i])
		}
	}
}

func TestRingQueueOverwrite(t *testing.T) {
	rq := NewRingQueue[int](3)

	// 填满队列
	rq.Push(1)
	rq.Push(2)
	rq.Push(3)

	// 继续添加，应该覆盖最老的数据
	rq.Push(4)
	rq.Push(5)

	data := rq.Bytes()
	// 应该包含最新的3个元素：3, 4, 5
	expected := []int{3, 4, 5}
	if len(data) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(data))
	}
	for i, v := range expected {
		if data[i] != v {
			t.Errorf("Expected %d at index %d, got %d", v, i, data[i])
		}
	}
}

func TestRingQueueConcurrentPush(t *testing.T) {
	rq := NewRingQueue[int](1000)
	var wg sync.WaitGroup

	// 启动多个 goroutine 并发写入
	numWriters := 10
	itemsPerWriter := 100

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < itemsPerWriter; j++ {
				rq.Push(start*itemsPerWriter + j)
			}
		}(i)
	}

	wg.Wait()

	// 验证队列状态
	if rq.Len() != 1000 {
		t.Errorf("Expected length 1000, got %d", rq.Len())
	}
	if !rq.IsFull() {
		t.Error("Expected queue to be full")
	}
}

func TestRingQueueConcurrentReadWrite(t *testing.T) {
	rq := NewRingQueue[int](100)
	var wg sync.WaitGroup

	// 写入者
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			rq.Push(i)
			time.Sleep(time.Microsecond)
		}
	}()

	// 多个读取者
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				data := rq.Bytes()
				_ = data // 允许脏读，只要不崩溃就行
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
}

func TestRingQueueClear(t *testing.T) {
	rq := NewRingQueue[int](5)

	// 添加数据
	for i := 1; i <= 5; i++ {
		rq.Push(i)
	}

	// 清空
	rq.Clear()

	if !rq.IsEmpty() {
		t.Error("Expected queue to be empty after clear")
	}
	if rq.Len() != 0 {
		t.Errorf("Expected length 0, got %d", rq.Len())
	}

	data := rq.Bytes()
	if data != nil {
		t.Error("Expected nil slice after clear")
	}
}

// 基准测试
func BenchmarkRingQueuePush(b *testing.B) {
	rq := NewRingQueue[int](1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rq.Push(i)
	}
}

func BenchmarkRingQueueBytes(b *testing.B) {
	rq := NewRingQueue[int](1000)

	// 预填充
	for i := 0; i < 1000; i++ {
		rq.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rq.Bytes()
	}
}

func BenchmarkRingQueueConcurrentPush(b *testing.B) {
	rq := NewRingQueue[int](10000)
	numCPU := runtime.NumCPU()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			rq.Push(i)
			i++
		}
	})

	_ = numCPU // 避免未使用变量警告
}

func BenchmarkRingQueueConcurrentRead(b *testing.B) {
	rq := NewRingQueue[int](1000)

	// 预填充
	for i := 0; i < 1000; i++ {
		rq.Push(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rq.Bytes()
		}
	})
}
