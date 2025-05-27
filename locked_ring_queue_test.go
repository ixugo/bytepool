package bytepool

import (
	"sync"
	"testing"
)

func TestLockedRingQueue_Basic(t *testing.T) {
	queue := NewLockedRingQueue[int](3)

	// Test initial state
	if !queue.IsEmpty() {
		t.Error("new queue should be empty")
	}
	if queue.IsFull() {
		t.Error("new queue should not be full")
	}
	if queue.Len() != 0 {
		t.Error("new queue length should be 0")
	}
	if queue.Cap() != 3 {
		t.Error("queue capacity should be 3")
	}

	// Test push and basic operations
	queue.Push(1)
	if queue.IsEmpty() {
		t.Error("queue should not be empty after push")
	}
	if queue.Len() != 1 {
		t.Error("queue length should be 1")
	}

	queue.Push(2)
	queue.Push(3)
	if !queue.IsFull() {
		t.Error("queue should be full")
	}
	if queue.Len() != 3 {
		t.Error("queue length should be 3")
	}

	// Test bytes
	bytes := queue.Bytes()
	expected := []int{1, 2, 3}
	if len(bytes) != len(expected) {
		t.Errorf("bytes length mismatch: got %d, want %d", len(bytes), len(expected))
	}
	for i, v := range expected {
		if bytes[i] != v {
			t.Errorf("bytes[%d] = %d, want %d", i, bytes[i], v)
		}
	}
}

func TestLockedRingQueue_PopAndPeek(t *testing.T) {
	queue := NewLockedRingQueue[string](3)

	// Test pop from empty queue
	if item, ok := queue.Pop(); ok {
		t.Errorf("pop from empty queue should return false, got %v", item)
	}

	// Test peek from empty queue
	if item, ok := queue.Peek(); ok {
		t.Errorf("peek from empty queue should return false, got %v", item)
	}

	// Add items
	queue.Push("a")
	queue.Push("b")
	queue.Push("c")

	// Test peek
	if item, ok := queue.Peek(); !ok || item != "a" {
		t.Errorf("peek should return 'a', got %v, %v", item, ok)
	}
	if queue.Len() != 3 {
		t.Error("peek should not change queue length")
	}

	// Test pop
	if item, ok := queue.Pop(); !ok || item != "a" {
		t.Errorf("pop should return 'a', got %v, %v", item, ok)
	}
	if queue.Len() != 2 {
		t.Error("queue length should be 2 after pop")
	}

	if item, ok := queue.Pop(); !ok || item != "b" {
		t.Errorf("pop should return 'b', got %v, %v", item, ok)
	}

	if item, ok := queue.Pop(); !ok || item != "c" {
		t.Errorf("pop should return 'c', got %v, %v", item, ok)
	}

	if !queue.IsEmpty() {
		t.Error("queue should be empty after popping all items")
	}
}

func TestLockedRingQueue_Overflow(t *testing.T) {
	queue := NewLockedRingQueue[int](3)

	// Fill the queue
	queue.Push(1)
	queue.Push(2)
	queue.Push(3)

	// Overflow - should overwrite oldest
	queue.Push(4)
	if queue.Len() != 3 {
		t.Error("queue length should remain 3 after overflow")
	}

	bytes := queue.Bytes()
	expected := []int{2, 3, 4}
	for i, v := range expected {
		if bytes[i] != v {
			t.Errorf("after overflow, bytes[%d] = %d, want %d", i, bytes[i], v)
		}
	}

	// Continue overflow
	queue.Push(5)
	queue.Push(6)

	bytes = queue.Bytes()
	expected = []int{4, 5, 6}
	for i, v := range expected {
		if bytes[i] != v {
			t.Errorf("after more overflow, bytes[%d] = %d, want %d", i, bytes[i], v)
		}
	}
}

func TestLockedRingQueue_Clear(t *testing.T) {
	queue := NewLockedRingQueue[int](3)

	queue.Push(1)
	queue.Push(2)
	queue.Push(3)

	queue.Clear()

	if !queue.IsEmpty() {
		t.Error("queue should be empty after clear")
	}
	if queue.Len() != 0 {
		t.Error("queue length should be 0 after clear")
	}
	if queue.Bytes() != nil {
		t.Error("bytes should be nil after clear")
	}
}

func TestLockedRingQueue_Concurrent(t *testing.T) {
	queue := NewLockedRingQueue[int](100)
	var wg sync.WaitGroup

	// Concurrent pushes
	numGoroutines := 10
	itemsPerGoroutine := 50

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(start int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				queue.Push(start*itemsPerGoroutine + j)
			}
		}(i)
	}

	wg.Wait()

	// Check that we have the expected number of items
	if queue.Len() != 100 {
		t.Errorf("expected queue length 100, got %d", queue.Len())
	}

	// Concurrent pops
	results := make([]int, 100)
	var mu sync.Mutex
	index := 0

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for {
				if item, ok := queue.Pop(); ok {
					mu.Lock()
					if index < len(results) {
						results[index] = item
						index++
					}
					mu.Unlock()
				} else {
					break
				}
			}
		}()
	}

	wg.Wait()

	if !queue.IsEmpty() {
		t.Error("queue should be empty after all pops")
	}
}

func TestLockedRingQueue_MixedOperations(t *testing.T) {
	queue := NewLockedRingQueue[int](5)

	// Mix push and pop operations
	queue.Push(1)
	queue.Push(2)

	if item, ok := queue.Pop(); !ok || item != 1 {
		t.Errorf("expected 1, got %v", item)
	}

	queue.Push(3)
	queue.Push(4)
	queue.Push(5)
	queue.Push(6) // should be at capacity

	if queue.Len() != 5 {
		t.Errorf("expected length 5, got %d", queue.Len())
	}

	bytes := queue.Bytes()
	expected := []int{2, 3, 4, 5, 6}
	for i, v := range expected {
		if bytes[i] != v {
			t.Errorf("bytes[%d] = %d, want %d", i, bytes[i], v)
		}
	}
}
