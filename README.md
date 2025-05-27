# BytePool - High-Performance Tiered Memory Pool with Reference Counting

English | [中文](readme_cn.md)

BytePool is a high-performance Go memory pool library designed to solve the "don't know when to release memory" scenario. It automatically manages memory lifecycle through reference counting mechanism, preventing memory leaks.

## 🚀 Key Features

- **Automatic Reference Counting**: No need to manually track memory release timing, automatically reclaimed when reference count reaches zero
- **Tiered Memory Pool**: Supports multi-tier memory allocation, reducing memory fragmentation and improving allocation efficiency
- **Zero-Copy Design**: Optimized based on `sync.Pool`, avoiding unnecessary memory copying
- **Thread-Safe**: Fully concurrent-safe, suitable for high-concurrency scenarios
- **Built-in Statistics**: Real-time monitoring of memory usage with expvar integration support
- **High Performance**: Memory allocation performance superior to direct `make([]byte, size)`, statistics add only 11ns overhead

## 🎯 Problems Solved

Common memory management challenges in Go development:

**When to use sync.Pool:**
If your scenario clearly knows when to release memory, we recommend using the standard library's `sync.Pool` directly. This library also provides a generic version of `sync.Pool` (`Pool[T]`) that offers better type safety, which you can try.

**Limitations of Traditional Solutions:**
- **Solution 1**: Independent memory pools per package → Memory waste, multiple copies
- **Solution 2**: Shared memory pool but unknown release timing → Memory leak risks

**BytePool's Solution:**
When you encounter "don't know when to release memory" scenarios, BytePool provides the perfect solution:
- Reference counting mechanism automatically manages memory lifecycle
- Tiered architecture optimizes allocation for different sizes
- Statistics help identify memory usage patterns and potential leaks

## 📦 Quick Start

### Installation

```bash
go get github.com/ixugo/bytepool
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/ixugo/bytepool"
)

func main() {
    // Create tiered memory pool (128B, 256B, 512B, 1KB, 2KB...)
    pool := bytepool.NewPools(bytepool.SizePowerOfTwo())

    // Method 1: Direct Get/Put usage
    buf := pool.Get(100) // Get at least 100 bytes of memory
    // Use buf...
    pool.Put(buf) // Manual release

    // Method 2: Use Buffer (Recommended)
    buffer := pool.GetBuffer(200) // Automatic reference counting management
    // Use buffer.Bytes()...
    buffer.Release() // Reference count-1, auto-reclaimed when reaches 0
}
```

### Reference Counting Example

There are two approaches for using reference counting:

**Approach 1: Manual Reference Counting**
```go
// Create Buffer
buffer := pool.GetBuffer(1024)

// Increase reference
buffer.Retain() // Reference count = 2

// Use in different goroutines
go func() {
    defer buffer.Release() // Reference count-1
    // Use buffer...
}()

// Main goroutine release
buffer.Release() // Reference count-1, now 0, auto-reclaimed to pool
```

**Approach 2: Automatic Management with Bytes()**
```go
// Create Buffer
buffer := pool.GetBuffer(1024)

// Use Bytes() method, automatically increases reference count and returns release function
data, release := buffer.Bytes()

// Use in different goroutines
go func() {
    defer release() // Automatically release reference
    // Use data...
}()

// Main goroutine release
buffer.Release() // Release initial reference
```

## 📊 Statistics

BytePool includes powerful built-in statistics to help monitor and optimize memory usage:

### View Statistics via expvar

```go
// Enable expvar statistics
pool.Expvar("myapp_")

// When using Go's default http service, you can access http://localhost:6060/debug/vars to view statistics
// Or get in code
stats := pool.GetPoolStats()
```

### Statistics Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `pools` | `map[int]map[string]int64` | Detailed statistics for each tier, key is tier size |
| `pools[size].get` | `int64` | Number of get operations for this tier |
| `pools[size].put` | `int64` | Number of put operations for this tier |
| `discarded` | `int64` | Number of allocations discarded (exceeding max tier) |
| `total_get` | `int64` | Total valid get operations (pool allocations only) |
| `total_put` | `int64` | Total valid put operations (pool reclaims only) |
| `recent_lengths` | `[]int` | Recent 256 get operation request lengths (ring buffer) |

### Memory Leak Detection

Compare `total_get` and `total_put` to detect potential memory leaks:

- `total_get > total_put`: Unreleased memory exists
- `total_get = total_put`: Memory allocation balanced
- Monitor `recent_lengths` to analyze memory allocation patterns

## 📚 API Documentation

### Core Types

#### BytePool

Main memory pool type that manages multi-tier memory allocation.

#### Buffer

Memory buffer with reference counting, automatically manages lifecycle.

#### Pool[T]

Generic version of `sync.Pool` that provides type-safe object pooling. If you clearly know when to release memory, you can use this type directly instead of BytePool.

### Creation Functions

#### `NewPools(sizes []int, opts ...Option) *BytePool`

Create a new tiered memory pool with optional configurations.

**Parameters:**
- `sizes`: Memory tier array, must be positive and ordered
- `opts`: Optional configuration functions

**Returns:**
- `*BytePool`: Memory pool instance

**Example:**
```go
// Use predefined power-of-2 tiers with default lock-free ring queue
pool := bytepool.NewPools(bytepool.SizePowerOfTwo())

// Custom tiers with mutex-based ring queue for strong consistency
pool := bytepool.NewPools([]int{128, 512, 2048},
    bytepool.WithRingQueueType(bytepool.MutexRingQueue))

// Use custom ring queue implementation
customQueue := bytepool.NewLockedRingQueue[int](512)
pool := bytepool.NewPools([]int{128, 512, 2048},
    bytepool.WithRingQueue(customQueue))
```

#### Ring Queue Options

BytePool supports two types of ring queues for tracking recent allocation lengths:

**Lock-Free Ring Queue (Default)**
- Uses atomic operations for maximum performance
- ~2.3x faster than mutex-based version in single-threaded scenarios
- ~1.6x faster in concurrent scenarios
- May have slight data inconsistency during concurrent access (acceptable for statistics)

**Mutex Ring Queue**
- Uses mutex locks for strong consistency guarantees
- Provides additional operations like `Pop()` and `Peek()`
- Better for scenarios requiring strict data consistency
- Slightly higher memory overhead due to additional tracking fields

**Performance Comparison:**

| Operation | Lock-Free | Mutex-Based | Performance Ratio |
|-----------|-----------|-------------|-------------------|
| Single Push | 8.2 ns/op | 19.1 ns/op | 2.3x faster |
| Concurrent Push | 63.9 ns/op | 102.0 ns/op | 1.6x faster |
| Bytes() | 1053 ns/op | 1666 ns/op | 1.6x faster |
| BytePool Integration | 186.1 ns/op | 291.8 ns/op | 1.6x faster |

**When to use each:**
- **Lock-Free (Default)**: High-performance scenarios, statistical monitoring, write-heavy workloads
- **Mutex-Based**: Strong consistency requirements, need Pop/Peek operations, debugging scenarios

#### `SizePowerOfTwo() []int`

Returns power-of-2 tier sequence (128B to 2MB).

#### `SizeStream() []int`

Returns tier sequence suitable for stream processing.

#### `NewPool[T any](fn func() T) *Pool[T]`

Create a new generic object pool.

**Parameters:**
- `fn`: Function to create new objects

**Returns:**
- `*Pool[T]`: Generic object pool instance

**Example:**
```go
// Create string pool
stringPool := bytepool.NewPool(func() string { return "" })

// Create custom struct pool
type MyStruct struct { Data []byte }
structPool := bytepool.NewPool(func() *MyStruct {
    return &MyStruct{Data: make([]byte, 1024)}
})
```

#### Ring Queue Creation Functions

#### `NewRingQueue[T any](size int) *RingQueue[T]`

Create a new lock-free ring queue.

**Parameters:**
- `size`: Queue capacity, must be positive

**Returns:**
- `*RingQueue[T]`: Lock-free ring queue instance

#### `NewLockedRingQueue[T any](size int) *LockedRingQueue[T]`

Create a new mutex-based ring queue.

**Parameters:**
- `size`: Queue capacity, must be positive

**Returns:**
- `*LockedRingQueue[T]`: Mutex-based ring queue instance

**Example:**
```go
// Create lock-free ring queue for high performance
lockFreeQueue := bytepool.NewRingQueue[int](256)

// Create mutex-based ring queue for strong consistency
lockedQueue := bytepool.NewLockedRingQueue[string](128)
```

#### Configuration Options

#### `WithRingQueueType(queueType RingQueueType) Option`

Configure the type of ring queue to use.

**Parameters:**
- `queueType`: Either `LockFreeRingQueue` (default) or `MutexRingQueue`

#### `WithRingQueue(ringQueuer RingQueuer) Option`

Use a custom ring queue implementation.

**Parameters:**
- `ringQueuer`: Custom ring queue that implements the `RingQueuer` interface

### Memory Allocation Methods

#### `(*BytePool) Get(length int) []byte`

Get byte slice of specified length from pool.

**Parameters:**
- `length`: Required number of bytes

**Returns:**
- `[]byte`: Byte slice with length, capacity may be larger

#### `(*BytePool) Put(buf []byte)`

Return byte slice to pool.

**Parameters:**
- `buf`: Byte slice to reclaim

#### `(*BytePool) GetBuffer(length int) *Buffer`

Get Buffer with reference counting.

**Parameters:**
- `length`: Required number of bytes

**Returns:**
- `*Buffer`: Buffer instance with initial reference count of 1

#### `(*BytePool) ReleaseBuffer(buf *Buffer)`

Release Buffer (equivalent to `buf.Release()`).

### Buffer Methods

#### `(*Buffer) Bytes() ([]byte, func())`

Get byte slice of Buffer, automatically increases reference count.

**Returns:**
- `[]byte`: Byte slice
- `func()`: Release function, automatically decreases reference count when called

#### `(*Buffer) Retain() *Buffer`

Increase reference count.

**Returns:**
- `*Buffer`: Returns self, supports method chaining

#### `(*Buffer) Release()`

Decrease reference count, auto-reclaimed to pool when reaches 0.

#### `(*Buffer) RefCount() int32`

Get current reference count.

### Statistics Methods

#### `(*BytePool) GetPoolStats() map[string]interface{}`

Get detailed pool statistics.

#### `(*BytePool) GetDiscardedCount() int64`

Get number of discarded allocations.

#### `(*BytePool) GetAvailableSizes() []int`

Get all available tier sizes.

#### `(*BytePool) Expvar(prefix string) *BytePool`

Enable expvar statistics with specified prefix.

**Parameters:**
- `prefix`: Prefix for statistics variables

**Returns:**
- `*BytePool`: Returns self, supports method chaining

### Pool[T] Methods

#### `(*Pool[T]) Get() T`

Get object from pool.

**Returns:**
- `T`: Object from pool or newly created object

#### `(*Pool[T]) Put(x T)`

Put object back to pool.

**Parameters:**
- `x`: Object to put back to pool

**Example:**
```go
pool := bytepool.NewPool(func() []byte { return make([]byte, 1024) })

// Get object
buf := pool.Get()
// Use buf...

// Put back to pool
pool.Put(buf)
```

### Ring Queue Methods

Both `RingQueue[T]` and `LockedRingQueue[T]` implement the `RingQueuer` interface and provide the following methods:

#### Common Methods (Both Types)

#### `Push(item T)`
Add an element to the queue. If the queue is full, overwrites the oldest element.

#### `Bytes() []T`
Returns all current data in the queue in order (oldest to newest).

#### `Len() int`
Returns the current number of elements in the queue.

#### `Cap() int`
Returns the queue capacity.

#### `IsFull() bool`
Checks if the queue is full.

#### `IsEmpty() bool`
Checks if the queue is empty.

#### `Clear()`
Empties the queue.

#### Additional Methods (LockedRingQueue Only)

#### `Pop() (T, bool)`
Removes and returns the oldest element from the queue. Returns zero value and false if queue is empty.

#### `Peek() (T, bool)`
Returns the oldest element without removing it. Returns zero value and false if queue is empty.

**Example:**
```go
// Lock-free ring queue usage
queue := bytepool.NewRingQueue[int](5)
queue.Push(1)
queue.Push(2)
queue.Push(3)

data := queue.Bytes() // [1, 2, 3]
fmt.Printf("Length: %d, Capacity: %d\n", queue.Len(), queue.Cap())

// Mutex-based ring queue with additional operations
lockedQueue := bytepool.NewLockedRingQueue[string](3)
lockedQueue.Push("a")
lockedQueue.Push("b")

if item, ok := lockedQueue.Peek(); ok {
    fmt.Printf("Oldest item: %s\n", item) // "a"
}

if item, ok := lockedQueue.Pop(); ok {
    fmt.Printf("Popped: %s\n", item) // "a"
}
```

## 🔧 Advanced Usage

### Custom Tier Configuration

Configure appropriate tiers based on application characteristics:

```go
// Optimized for small objects
smallPool := bytepool.NewPools([]int{64, 128, 256, 512})

// Optimized for large file processing
largePool := bytepool.NewPools([]int{4096, 16384, 65536, 262144})
```

### Performance Monitoring

```go
// Periodically check memory usage
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        stats := pool.GetPoolStats()
        totalGet := stats["total_get"].(int64)
        totalPut := stats["total_put"].(int64)

        if totalGet != totalPut {
            log.Printf("Memory leak warning: get=%d, put=%d", totalGet, totalPut)
        }
    }
}()
```

### Using Generic Object Pool

When you clearly know when to release memory, you can use generic Pool[T]:

```go
// Create buffer pool
bufferPool := bytepool.NewPool(func() *bytes.Buffer {
    return &bytes.Buffer{}
})

// Usage
buf := bufferPool.Get()
buf.Reset() // Clear buffer
buf.WriteString("Hello, World!")
// Put back when done
bufferPool.Put(buf)

// Create custom struct pool
type Request struct {
    ID   int
    Data []byte
}

requestPool := bytepool.NewPool(func() *Request {
    return &Request{Data: make([]byte, 1024)}
})
```

## 📈 Performance Comparison

| Operation | BytePool | Direct Allocation | Performance Gain |
|-----------|----------|-------------------|------------------|
| Small object allocation (128B) | 88.94 ns/op | 532.5 ns/op | ~6x |
| Large object allocation (64KB) | 761.0 ns/op | - | - |
| Statistics overhead | +11 ns/op | - | Minimal |

## 🤝 Contributing

Issues and Pull Requests are welcome!

## 📄 License

MIT License