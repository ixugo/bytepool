package bytepool

import (
	"sync"
	"testing"
)

func TestBytePool_RecentLengthsTracking(t *testing.T) {
	pool := NewPools([]int{128, 256, 512, 1024})

	// 测试基本统计功能
	lengths := []int{100, 200, 300, 150, 250}
	for _, length := range lengths {
		buf := pool.Get(length)
		pool.Put(buf)
	}

	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)
	totalPut := stats["total_put"].(int64)

	if len(recentLengths) != len(lengths) {
		t.Errorf("Expected recent lengths count %d, got %d", len(lengths), len(recentLengths))
	}

	if totalGet != int64(len(lengths)) {
		t.Errorf("Expected total_get %d, got %d", len(lengths), totalGet)
	}

	if totalPut != int64(len(lengths)) {
		t.Errorf("Expected total_put %d, got %d", len(lengths), totalPut)
	}

	for i, expected := range lengths {
		if recentLengths[i] != expected {
			t.Errorf("Expected recent length[%d] = %d, got %d", i, expected, recentLengths[i])
		}
	}
}

func TestBytePool_RecentLengthsOverflow(t *testing.T) {
	pool := NewPools([]int{128, 256, 512, 1024})

	// 写入超过 256 个长度，测试环形队列的循环覆盖
	for i := 1; i <= 300; i++ {
		pool.Get(i)
	}

	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)

	// 应该只保留最近的 256 个
	if len(recentLengths) != 256 {
		t.Errorf("Expected recent lengths count 256, got %d", len(recentLengths))
	}

	// 总的 get 操作应该是 300
	if totalGet != 300 {
		t.Errorf("Expected total_get 300, got %d", totalGet)
	}

	// 验证是最新的 256 个（45-300）
	for i, length := range recentLengths {
		expected := 45 + i // 从第 45 个开始（300-256+1=45）
		if length != expected {
			t.Errorf("Expected recent length[%d] = %d, got %d", i, expected, length)
		}
	}
}

func TestBytePool_GetBufferBalance(t *testing.T) {
	pool := NewPools([]int{128, 256, 512, 1024})

	// 测试 GetBuffer 和 ReleaseBuffer 的平衡
	buffers := make([]*Buffer, 10)
	for i := 0; i < 10; i++ {
		buffers[i] = pool.GetBuffer(100)
	}

	stats := pool.GetPoolStats()
	totalGet := stats["total_get"].(int64)
	totalPut := stats["total_put"].(int64)

	if totalGet != 10 {
		t.Errorf("Expected total_get 10, got %d", totalGet)
	}

	if totalPut != 0 {
		t.Errorf("Expected total_put 0, got %d", totalPut)
	}

	// 释放一半
	for i := 0; i < 5; i++ {
		pool.ReleaseBuffer(buffers[i])
	}

	stats = pool.GetPoolStats()
	totalPut = stats["total_put"].(int64)

	if totalPut != 5 {
		t.Errorf("Expected total_put 5, got %d", totalPut)
	}

	// 释放剩余的
	for i := 5; i < 10; i++ {
		pool.ReleaseBuffer(buffers[i])
	}

	stats = pool.GetPoolStats()
	totalPut = stats["total_put"].(int64)

	if totalPut != 10 {
		t.Errorf("Expected total_put 10, got %d", totalPut)
	}
}

func TestBytePool_OnlyValidOperationsTracked(t *testing.T) {
	pool := NewPools([]int{128, 256, 512})

	// 测试超过最大档位的操作不被统计
	buf1 := pool.Get(1000) // 超过最大档位 512
	buf2 := pool.Get(100)  // 正常档位

	stats := pool.GetPoolStats()
	totalGet := stats["total_get"].(int64)
	recentLengths := stats["recent_lengths"].([]int)

	// 只有有效的操作被统计
	if totalGet != 1 {
		t.Errorf("Expected total_get 1, got %d", totalGet)
	}

	// 环形队列记录所有操作（包括无效的）
	if len(recentLengths) != 2 {
		t.Errorf("Expected recent_lengths count 2, got %d", len(recentLengths))
	}

	if recentLengths[0] != 1000 || recentLengths[1] != 100 {
		t.Errorf("Expected recent_lengths [1000 100], got %v", recentLengths)
	}

	// Put 操作
	pool.Put(buf1) // 超过最大档位，不会被统计
	pool.Put(buf2) // 正常档位

	stats = pool.GetPoolStats()
	totalPut := stats["total_put"].(int64)

	if totalPut != 1 {
		t.Errorf("Expected total_put 1, got %d", totalPut)
	}
}

func TestBytePool_RecentLengthsConcurrent(t *testing.T) {
	pool := NewPools([]int{128, 256, 512, 1024})

	// 并发写入测试
	const numGoroutines = 10
	const numOpsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				length := base*10 + j + 1 // 确保在有效范围内
				buf := pool.Get(length)
				pool.Put(buf)
			}
		}(i)
	}

	wg.Wait()

	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)
	totalPut := stats["total_put"].(int64)

	// 应该记录了 256 个长度（因为总共 500 个操作）
	if len(recentLengths) != 256 {
		t.Errorf("Expected recent lengths count 256, got %d", len(recentLengths))
	}

	// 总操作数应该是 500
	if totalGet != 500 {
		t.Errorf("Expected total_get 500, got %d", totalGet)
	}

	if totalPut != 500 {
		t.Errorf("Expected total_put 500, got %d", totalPut)
	}

	// 验证所有记录的长度都是正数
	for i, length := range recentLengths {
		if length <= 0 {
			t.Errorf("Invalid length at index %d: %d", i, length)
		}
	}
}
