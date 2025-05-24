# BytePool - 带引用计数的高性能分层内存池

[English](README.md) | 中文

BytePool 是一个高性能的 Go 语言内存池库，专为解决"不知道何时释放内存"的场景而设计。通过引用计数机制，自动管理内存生命周期，避免内存泄漏。

## 🚀 核心亮点

- **引用计数自动管理**：无需手动追踪内存释放时机，引用计数归零时自动回收
- **分层内存池**：支持多档位内存分配，减少内存碎片，提高分配效率
- **零拷贝设计**：基于 `sync.Pool` 优化，避免不必要的内存拷贝
- **线程安全**：完全并发安全，适用于高并发场景
- **内置统计**：实时监控内存使用情况，支持 expvar 集成
- **高性能**：内存分配性能优于直接 `make([]byte, size)`，统计功能仅增加 11ns 开销

## 🎯 解决的问题

在 Go 开发中，经常遇到以下内存管理难题：

**何时使用 sync.Pool：**
如果你的场景中明确知道何时释放内存，建议直接使用标准库的 `sync.Pool`。本库也提供了泛型版本的 `sync.Pool`（`Pool[T]`），可以提供更好的类型安全性，你可以尝试使用。

**传统方案的局限性：**
- **方案一**：各包独立内存池 → 内存浪费，多次拷贝
- **方案二**：共享内存池但不知何时释放 → 内存泄漏风险

**BytePool 的解决方案：**
当你遇到"不知道何时释放内存"的场景时，BytePool 提供了完美的解决方案：
- 引用计数机制自动管理内存生命周期
- 分层架构优化不同大小的内存分配
- 统计功能帮助识别内存使用模式和潜在泄漏

## 📦 快速开始

### 安装

```bash
go get -u github.com/ixugo/bytepool
```

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/ixugo/bytepool"
)

func main() {
    // 创建分层内存池（128B, 256B, 512B, 1KB, 2KB...）
    pool := bytepool.NewPools(bytepool.SizePowerOfTwo())

    // 方式一：直接使用 Get/Put
    buf := pool.Get(100) // 获取至少 100 字节的内存
    // 使用 buf...
    pool.Put(buf) // 手动释放

    // 方式二：使用 Buffer（推荐）
    buffer := pool.GetBuffer(200) // 自动引用计数管理
    // 使用 buffer.Bytes()...
    buffer.Release() // 引用计数-1，为0时自动回收
}
```

### 引用计数示例

引用计数有两种使用方案：

**方案一：手动管理引用计数**
```go
// 创建 Buffer
buffer := pool.GetBuffer(1024)

// 增加引用
buffer.Retain() // 引用计数 = 2

// 在不同 goroutine 中使用
go func() {
    defer buffer.Release() // 引用计数-1
    // 使用 buffer...
}()

// 主 goroutine 释放
buffer.Release() // 引用计数-1，此时为0，自动回收到池中
```

**方案二：使用 Bytes() 自动管理**
```go
// 创建 Buffer
buffer := pool.GetBuffer(1024)

// 使用 Bytes() 方法，自动增加引用计数并返回释放函数
data, release := buffer.Bytes()

// 在不同 goroutine 中使用
go func() {
    defer release() // 自动释放引用
    // 使用 data...
}()

// 主 goroutine 释放
buffer.Release() // 释放初始引用
```

## 📊 统计功能

BytePool 内置强大的统计功能，帮助监控和优化内存使用：

### 通过 expvar 查看统计数据

```go
// 启用 expvar 统计
pool.Expvar("myapp_")

// 使用 Go 默认的 http 服务时，可以访问 http://localhost:6060/debug/vars 查看统计数据
// 或在代码中获取
stats := pool.GetPoolStats()
```

### 统计参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `pools` | `map[int]map[string]int64` | 各档位的详细统计，key为档位大小 |
| `pools[size].get` | `int64` | 该档位的 get 操作次数 |
| `pools[size].put` | `int64` | 该档位的 put 操作次数 |
| `discarded` | `int64` | 超过最大档位被丢弃的分配次数 |
| `total_get` | `int64` | 总的有效 get 操作数量（仅统计内存池分配） |
| `total_put` | `int64` | 总的有效 put 操作数量（仅统计内存池回收） |
| `recent_lengths` | `[]int` | 最近 256 个 get 操作的请求长度（环形队列） |

### 内存泄漏检测

通过比较 `total_get` 和 `total_put` 可以检测潜在的内存泄漏：

- `total_get > total_put`：存在未释放的内存
- `total_get = total_put`：内存分配平衡
- 监控 `recent_lengths` 可分析内存分配模式

## 📚 API 文档

### 核心类型

#### BytePool

主要的内存池类型，管理多个档位的内存分配。

#### Buffer

带引用计数的内存缓冲区，自动管理生命周期。

#### Pool[T]

泛型版本的 `sync.Pool`，提供类型安全的对象池。如果你明确知道何时释放内存，可以直接使用这个类型而不是 BytePool。

### 创建函数

#### `NewPools(sizes []int) *BytePool`

创建新的分层内存池。

**参数：**
- `sizes`: 内存档位数组，必须为正数且有序

**返回：**
- `*BytePool`: 内存池实例

**示例：**
```go
// 使用预定义的 2 的幂次方档位
pool := bytepool.NewPools(bytepool.SizePowerOfTwo())

// 自定义档位
pool := bytepool.NewPools([]int{128, 512, 2048})
```

#### `SizePowerOfTwo() []int`

返回 2 的幂次方档位序列（128B 到 2MB）。

#### `SizeStream() []int`

返回适合流处理的档位序列。

#### `NewPool[T any](fn func() T) *Pool[T]`

创建新的泛型对象池。

**参数：**
- `fn`: 创建新对象的函数

**返回：**
- `*Pool[T]`: 泛型对象池实例

**示例：**
```go
// 创建字符串池
stringPool := bytepool.NewPool(func() string { return "" })

// 创建自定义结构体池
type MyStruct struct { Data []byte }
structPool := bytepool.NewPool(func() *MyStruct {
    return &MyStruct{Data: make([]byte, 1024)}
})
```

### 内存分配方法

#### `(*BytePool) Get(length int) []byte`

从池中获取指定长度的字节切片。

**参数：**
- `length`: 需要的字节数

**返回：**
- `[]byte`: 字节切片，长度为 length，容量可能更大

#### `(*BytePool) Put(buf []byte)`

将字节切片放回池中。

**参数：**
- `buf`: 要回收的字节切片

#### `(*BytePool) GetBuffer(length int) *Buffer`

获取带引用计数的 Buffer。

**参数：**
- `length`: 需要的字节数

**返回：**
- `*Buffer`: Buffer 实例，初始引用计数为 1

#### `(*BytePool) ReleaseBuffer(buf *Buffer)`

释放 Buffer（等同于 `buf.Release()`）。

### Buffer 方法

#### `(*Buffer) Bytes() ([]byte, func())`

获取 Buffer 的字节切片，自动增加引用计数。

**返回：**
- `[]byte`: 字节切片
- `func()`: 释放函数，调用时自动减少引用计数

#### `(*Buffer) Retain() *Buffer`

增加引用计数。

**返回：**
- `*Buffer`: 返回自身，支持链式调用

#### `(*Buffer) Release()`

减少引用计数，为 0 时自动回收到池中。

#### `(*Buffer) RefCount() int32`

获取当前引用计数。

### 统计方法

#### `(*BytePool) GetPoolStats() map[string]interface{}`

获取详细的池统计信息。

#### `(*BytePool) GetDiscardedCount() int64`

获取被丢弃的分配次数。

#### `(*BytePool) GetAvailableSizes() []int`

获取所有可用的档位大小。

#### `(*BytePool) Expvar(prefix string) *BytePool`

启用 expvar 统计，使用指定前缀。

**参数：**
- `prefix`: 统计变量的前缀

**返回：**
- `*BytePool`: 返回自身，支持链式调用

### Pool[T] 方法

#### `(*Pool[T]) Get() T`

从池中获取对象。

**返回：**
- `T`: 池中的对象或新创建的对象

#### `(*Pool[T]) Put(x T)`

将对象放回池中。

**参数：**
- `x`: 要放回池中的对象

**示例：**
```go
pool := bytepool.NewPool(func() []byte { return make([]byte, 1024) })

// 获取对象
buf := pool.Get()
// 使用 buf...

// 放回池中
pool.Put(buf)
```

## 🔧 高级用法

### 自定义档位配置

根据应用特点配置合适的档位：

```go
// 针对小对象优化
smallPool := bytepool.NewPools([]int{64, 128, 256, 512})

// 针对大文件处理优化
largePool := bytepool.NewPools([]int{4096, 16384, 65536, 262144})
```

### 使用泛型对象池

当你明确知道何时释放内存时，可以使用泛型 Pool[T]：

```go
// 创建缓冲区池
bufferPool := bytepool.NewPool(func() *bytes.Buffer {
    return &bytes.Buffer{}
})

// 使用
buf := bufferPool.Get()
buf.Reset() // 清空缓冲区
buf.WriteString("Hello, World!")
// 使用完毕后放回
bufferPool.Put(buf)

// 创建自定义结构体池
type Request struct {
    ID   int
    Data []byte
}

requestPool := bytepool.NewPool(func() *Request {
    return &Request{Data: make([]byte, 1024)}
})
```

### 性能监控

```go
// 定期检查内存使用情况
go func() {
    ticker := time.NewTicker(time.Minute)
    for range ticker.C {
        stats := pool.GetPoolStats()
        totalGet := stats["total_get"].(int64)
        totalPut := stats["total_put"].(int64)

        if totalGet != totalPut {
            log.Printf("内存泄漏警告: get=%d, put=%d", totalGet, totalPut)
        }
    }
}()
```

## 📈 性能对比

| 操作 | BytePool | 直接分配 | 性能提升 |
|------|----------|----------|----------|
| 小对象分配 (128B) | 88.94 ns/op | 532.5 ns/op | ~6x |
| 大对象分配 (64KB) | 761.0 ns/op | - | - |
| 统计开销 | +11 ns/op | - | 极小 |

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
