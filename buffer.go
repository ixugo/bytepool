package bytepool

import (
	"sync/atomic"
)

type Buffer struct {
	buf      atomic.Pointer[[]byte] // 使用类型安全的atomic.Pointer
	refCount int32
	pools    *BytePool
}

func (b *Buffer) Bytes() ([]byte, func()) {
	b.Retain()

	// 原子读取buf指针
	bufPtr := b.buf.Load()
	if bufPtr == nil {
		return nil, func() {}
	}

	return *bufPtr, b.Release
}

func (b *Buffer) Release() {
	if atomic.AddInt32(&b.refCount, -1) == 0 {
		bufPtr := b.buf.Swap(nil)
		if bufPtr != nil {
			b.pools.Put(*bufPtr)
		}
	}
}

func (b *Buffer) Retain() {
	atomic.AddInt32(&b.refCount, 1)
}

func NewBuffer(data []byte, pools *BytePool) *Buffer {
	buf := &Buffer{
		refCount: 1,
		pools:    pools,
	}
	buf.buf.Store(&data)
	return buf
}
