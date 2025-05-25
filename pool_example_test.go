package bytepool

import (
	"encoding/json"
	"fmt"
)

// Example_recentLengthsTracking demonstrates the recent 256 get operation length tracking feature
func Example_recentLengthsTracking() {
	// create a BytePool
	pool := NewPools([]int{128, 256, 512, 1024})

	// simulate some get operations
	lengths := []int{100, 200, 50, 300, 150, 400, 80}

	fmt.Println("Simulating get operations:")
	for i, length := range lengths {
		buf := pool.Get(length)
		fmt.Printf("Get(%d) -> allocated %d bytes\n", length, len(buf))
		pool.Put(buf)

		// check statistics every few operations
		if i == 2 || i == len(lengths)-1 {
			stats := pool.GetPoolStats()
			recentLengths := stats["recent_lengths"].([]int)
			totalGet := stats["total_get"].(int64)
			totalPut := stats["total_put"].(int64)
			fmt.Printf("Current recorded lengths: %v (total get:%d, total put:%d)\n",
				recentLengths, totalGet, totalPut)
		}
	}

	// final statistics
	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)
	totalPut := stats["total_put"].(int64)

	fmt.Printf("\nFinal statistics:\n")
	fmt.Printf("Recorded operations: %d\n", len(recentLengths))
	fmt.Printf("Total get operations: %d\n", totalGet)
	fmt.Printf("Total put operations: %d\n", totalPut)
	fmt.Printf("Recent lengths: %v\n", recentLengths)

	// Output:
	// Simulating get operations:
	// Get(100) -> allocated 100 bytes
	// Get(200) -> allocated 200 bytes
	// Get(50) -> allocated 50 bytes
	// Current recorded lengths: [100 200 50] (total get:3, total put:3)
	// Get(300) -> allocated 300 bytes
	// Get(150) -> allocated 150 bytes
	// Get(400) -> allocated 400 bytes
	// Get(80) -> allocated 80 bytes
	// Current recorded lengths: [100 200 50 300 150 400 80] (total get:7, total put:7)
	//
	// Final statistics:
	// Recorded operations: 7
	// Total get operations: 7
	// Total put operations: 7
	// Recent lengths: [100 200 50 300 150 400 80]
}

// Example_recentLengthsOverflow demonstrates ring buffer overflow when exceeding 256 operations
func Example_recentLengthsOverflow() {
	pool := NewPools([]int{128, 256, 512, 1024})

	// simulate many operations (more than 256)
	fmt.Println("Executing 260 get operations...")
	for i := 1; i <= 260; i++ {
		pool.Get(i)
	}

	stats := pool.GetPoolStats()
	recentLengths := stats["recent_lengths"].([]int)
	totalGet := stats["total_get"].(int64)

	fmt.Printf("Total operations: 260\n")
	fmt.Printf("Recorded operations: %d (max 256)\n", len(recentLengths))
	fmt.Printf("Total get statistics: %d\n", totalGet)
	fmt.Printf("Earliest recorded length: %d (should be 5th operation)\n", recentLengths[0])
	fmt.Printf("Latest recorded length: %d (should be 260th operation)\n", recentLengths[len(recentLengths)-1])

	// Output:
	// Executing 260 get operations...
	// Total operations: 260
	// Recorded operations: 256 (max 256)
	// Total get statistics: 260
	// Earliest recorded length: 5 (should be 5th operation)
	// Latest recorded length: 260 (should be 260th operation)
}

// Example_poolStatsWithRecentLengths demonstrates complete statistics information
func Example_poolStatsWithRecentLengths() {
	pool := NewPools([]int{128, 256, 512})

	// perform some operations
	for i := 0; i < 10; i++ {
		buf := pool.Get(100 + i*10)
		pool.Put(buf)
	}

	// get complete statistics
	stats := pool.GetPoolStats()

	// formatted output
	jsonData, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Printf("BytePool statistics:\n%s\n", jsonData)

	// show recent length statistics separately
	recentLengths := stats["recent_lengths"].([]int)
	fmt.Printf("\nRecent request length analysis:\n")
	fmt.Printf("Total requests: %d\n", len(recentLengths))

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
		fmt.Printf("Min length: %d\n", min)
		fmt.Printf("Max length: %d\n", max)
		fmt.Printf("Average length: %.1f\n", avg)
	}
}

// Example_bufferBalance demonstrates GetBuffer and ReleaseBuffer statistics
func Example_bufferBalance() {
	pool := NewPools([]int{128, 256, 512})

	fmt.Println("Creating 5 Buffers:")
	buffers := make([]*Buffer, 5)
	for i := 0; i < 5; i++ {
		buffers[i] = pool.GetBuffer(100)
		fmt.Printf("Created Buffer %d\n", i+1)
	}

	stats := pool.GetPoolStats()
	fmt.Printf("Statistics after creation: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	fmt.Println("\nReleasing 3 Buffers:")
	for i := 0; i < 3; i++ {
		pool.ReleaseBuffer(buffers[i])
		fmt.Printf("Released Buffer %d\n", i+1)
	}

	stats = pool.GetPoolStats()
	fmt.Printf("Statistics after partial release: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	fmt.Println("\nReleasing remaining 2 Buffers:")
	for i := 3; i < 5; i++ {
		pool.ReleaseBuffer(buffers[i])
		fmt.Printf("Released Buffer %d\n", i+1)
	}

	stats = pool.GetPoolStats()
	fmt.Printf("Statistics after full release: get=%d, put=%d\n",
		stats["total_get"].(int64), stats["total_put"].(int64))

	// Output:
	// Creating 5 Buffers:
	// Created Buffer 1
	// Created Buffer 2
	// Created Buffer 3
	// Created Buffer 4
	// Created Buffer 5
	// Statistics after creation: get=5, put=0
	//
	// Releasing 3 Buffers:
	// Released Buffer 1
	// Released Buffer 2
	// Released Buffer 3
	// Statistics after partial release: get=5, put=3
	//
	// Releasing remaining 2 Buffers:
	// Released Buffer 4
	// Released Buffer 5
	// Statistics after full release: get=5, put=5
}
