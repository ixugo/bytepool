package bytepool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRingQueueBasic(t *testing.T) {
	rq := NewRingQueue[int](3)

	// test basic operations
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

	// fill the queue
	rq.Push(1)
	rq.Push(2)
	rq.Push(3)

	// continue adding, should overwrite oldest data
	rq.Push(4)
	rq.Push(5)

	data := rq.Bytes()
	// should contain the latest 3 elements: 3, 4, 5
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

	// start multiple goroutines for concurrent writing
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

	// verify queue state
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

	// writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			rq.Push(i)
			time.Sleep(time.Microsecond)
		}
	}()

	// multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				data := rq.Bytes()
				_ = data // allow dirty reads, as long as it doesn't crash
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
}

func TestRingQueueClear(t *testing.T) {
	rq := NewRingQueue[int](5)

	// add data
	for i := 1; i <= 5; i++ {
		rq.Push(i)
	}

	// clear
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

// benchmark tests
func BenchmarkRingQueuePush(b *testing.B) {
	rq := NewRingQueue[int](1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rq.Push(i)
	}
}

func BenchmarkRingQueueBytes(b *testing.B) {
	rq := NewRingQueue[int](1000)

	// pre-fill
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

	_ = numCPU // avoid unused variable warning
}

func BenchmarkRingQueueConcurrentRead(b *testing.B) {
	rq := NewRingQueue[int](1000)

	// pre-fill
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
