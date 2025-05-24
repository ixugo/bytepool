package bytepool

import (
	"encoding/json"
	"fmt"
)

// Example_recentLengthsTracking 展示最近 256 个 get 操作长度统计功能
func Example_recentLengthsTracking() {
	// 创建一个 BytePool
	pool := NewPools([]int{128, 256, 512, 1024})

	// 模拟一些 get 操作
	lengths := []int{100, 200, 50, 300, 150, 400, 80}

	fmt.Println("模拟 get 操作:")
	for i, length := range lengths {
		buf := pool.Get(length)
		fmt.Printf("Get(%d) -> 分配了 %d 字节\n", length, len(buf))
		pool.Put(buf)

		// 每隔几次操作查看统计
		if i == 2 || i == len(lengths)-1 {
			stats := pool.GetPoolStats()
			recentLengths := stats["recent_lengths"].([]int)
			totalGet := stats["total_get"].(int64)
			totalPut := stats["total_put"].(int64)
			fmt.Printf("当前记录的长度: %v (总get:%d, 总put:%d)\n",
				recentLengths, totalGet, totalPut)
		}
	}

	// 最终统计
	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)
	totalPut := stats["total_put"].(int64)

	fmt.Printf("\n最终统计:\n")
	fmt.Printf("记录的操作数: %d\n", len(recentLengths))
	fmt.Printf("总get操作: %d\n", totalGet)
	fmt.Printf("总put操作: %d\n", totalPut)
	fmt.Printf("最近的长度: %v\n", recentLengths)

	// Output:
	// 模拟 get 操作:
	// Get(100) -> 分配了 100 字节
	// Get(200) -> 分配了 200 字节
	// Get(50) -> 分配了 50 字节
	// 当前记录的长度: [100 200 50] (总get:3, 总put:3)
	// Get(300) -> 分配了 300 字节
	// Get(150) -> 分配了 150 字节
	// Get(400) -> 分配了 400 字节
	// Get(80) -> 分配了 80 字节
	// 当前记录的长度: [100 200 50 300 150 400 80] (总get:7, 总put:7)
	//
	// 最终统计:
	// 记录的操作数: 7
	// 总get操作: 7
	// 总put操作: 7
	// 最近的长度: [100 200 50 300 150 400 80]
}

// Example_recentLengthsOverflow 展示超过 256 个操作时的环形覆盖
func Example_recentLengthsOverflow() {
	pool := NewPools([]int{128, 256, 512, 1024})

	// 模拟大量操作（超过 256 个）
	fmt.Println("执行 260 个 get 操作...")
	for i := 1; i <= 260; i++ {
		pool.Get(i)
	}

	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)

	fmt.Printf("总操作数: 260\n")
	fmt.Printf("记录的操作数: %d (最大 256)\n", len(recentLengths))
	fmt.Printf("总get统计: %d\n", totalGet)
	fmt.Printf("最早记录的长度: %d (应该是第 5 个操作)\n", recentLengths[0])
	fmt.Printf("最新记录的长度: %d (应该是第 260 个操作)\n", recentLengths[len(recentLengths)-1])

	// Output:
	// 执行 260 个 get 操作...
	// 总操作数: 260
	// 记录的操作数: 256 (最大 256)
	// 总get统计: 260
	// 最早记录的长度: 5 (应该是第 5 个操作)
	// 最新记录的长度: 260 (应该是第 260 个操作)
}

// Example_poolStatsWithRecentLengths 展示完整的统计信息
func Example_poolStatsWithRecentLengths() {
	pool := NewPools([]int{128, 256, 512})

	// 执行一些操作
	for i := 0; i < 10; i++ {
		buf := pool.Get(100 + i*10)
		pool.Put(buf)
	}

	// 获取完整统计
	stats := pool.GetPoolStats()

	// 格式化输出
	jsonData, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Printf("BytePool 统计信息:\n%s\n", jsonData)

	// 单独展示最近长度统计
	recentLengths := stats["recent_lengths"].([]int)
	fmt.Printf("\n最近请求的长度分析:\n")
	fmt.Printf("总请求数: %d\n", len(recentLengths))

	if len(recentLengths) > 0 {
		min, max := recentLengths[0], recentLengths[0]
		sum := 0
		for _, length := range recentLengths {
			if length < min {
				min = length
			}
			if length > max {
				max = length
			}
			sum += length
		}
		avg := float64(sum) / float64(len(recentLengths))
		fmt.Printf("最小长度: %d\n", min)
		fmt.Printf("最大长度: %d\n", max)
		fmt.Printf("平均长度: %.1f\n", avg)
	}
}

// Example_bufferBalance 展示 GetBuffer 和 ReleaseBuffer 的统计
func Example_bufferBalance() {
	pool := NewPools([]int{128, 256, 512})

	fmt.Println("创建 5 个 Buffer:")
	buffers := make([]*Buffer, 5)
	for i := 0; i < 5; i++ {
		buffers[i] = pool.GetBuffer(100)
		fmt.Printf("创建 Buffer %d\n", i+1)
	}

	stats := pool.GetPoolStats()
	fmt.Printf("创建后统计: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	fmt.Println("\n释放 3 个 Buffer:")
	for i := 0; i < 3; i++ {
		pool.ReleaseBuffer(buffers[i])
		fmt.Printf("释放 Buffer %d\n", i+1)
	}

	stats = pool.GetPoolStats()
	fmt.Printf("部分释放后统计: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	fmt.Println("\n释放剩余 2 个 Buffer:")
	for i := 3; i < 5; i++ {
		pool.ReleaseBuffer(buffers[i])
		fmt.Printf("释放 Buffer %d\n", i+1)
	}

	stats = pool.GetPoolStats()
	fmt.Printf("全部释放后统计: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	// Output:
	// 创建 5 个 Buffer:
	// 创建 Buffer 1
	// 创建 Buffer 2
	// 创建 Buffer 3
	// 创建 Buffer 4
	// 创建 Buffer 5
	// 创建后统计: get=5, put=0
	//
	// 释放 3 个 Buffer:
	// 释放 Buffer 1
	// 释放 Buffer 2
	// 释放 Buffer 3
	// 部分释放后统计: get=5, put=3
	//
	// 释放剩余 2 个 Buffer:
	// 释放 Buffer 4
	// 释放 Buffer 5
	// 全部释放后统计: get=5, put=5
}
