package bytepool

import (
	"fmt"
)

// ExampleNewRingQueue 展示简化版无锁环形队列的使用
func ExampleNewRingQueue() {
	// 创建一个容量为 3 的环形队列
	rq := NewRingQueue[int](3)

	// 添加元素（未循环阶段）
	rq.Push(1)
	rq.Push(2)
	fmt.Printf("Before full: %v\n", rq.Bytes())

	rq.Push(3)
	fmt.Printf("Just full: %v\n", rq.Bytes())

	// 继续添加元素（循环阶段，会覆盖最老的数据）
	rq.Push(4)
	fmt.Printf("After cycle: %v\n", rq.Bytes())

	rq.Push(5)
	fmt.Printf("After cycle 2: %v\n", rq.Bytes())

	// Output:
	// Before full: [1 2]
	// Just full: [1 2 3]
	// After cycle: [2 3 4]
	// After cycle 2: [3 4 5]
}

// ExampleRingQueue_writeHeavy 展示写入密集型场景
func ExampleRingQueue_writeHeavy() {
	// 创建一个小容量的队列模拟写入密集场景
	rq := NewRingQueue[string](3)

	// 模拟大量写入
	messages := []string{"msg1", "msg2", "msg3", "msg4", "msg5", "msg6"}

	for i, msg := range messages {
		rq.Push(msg)
		fmt.Printf("After push %d: len=%d, data=%v\n", i+1, rq.Len(), rq.Bytes())
	}

	// Output:
	// After push 1: len=1, data=[msg1]
	// After push 2: len=2, data=[msg1 msg2]
	// After push 3: len=3, data=[msg1 msg2 msg3]
	// After push 4: len=3, data=[msg2 msg3 msg4]
	// After push 5: len=3, data=[msg3 msg4 msg5]
	// After push 6: len=3, data=[msg4 msg5 msg6]
}
