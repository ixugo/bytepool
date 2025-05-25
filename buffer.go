package bytepool

import (
	"sync/atomic"
)

// Buffer represents a reference-counted byte buffer that can be safely shared
type Buffer struct {
	buf      atomic.Pointer[[]byte] // use type-safe atomic.Pointer
	refCount int32
	pools    *BytePool
}

// Bytes returns the buffer data and a release function
// The caller must call the release function when done with the data
func (b *Buffer) Bytes() ([]byte, func()) {
	b.Retain()

	// atomically read buf pointer
	bufPtr := b.buf.Load()
	if bufPtr == nil {
		return nil, func() {}
	}

	return *bufPtr, b.Release
}

// Release decrements the reference count and returns the buffer to pool when count reaches zero
func (b *Buffer) Release() {
	if atomic.AddInt32(&b.refCount, -1) == 0 {
		bufPtr := b.buf.Swap(nil)
		if bufPtr != nil {
			b.pools.Put(*bufPtr)
		}
	}
}

// Retain increments the reference count
func (b *Buffer) Retain() {
	atomic.AddInt32(&b.refCount, 1)
}

// NewBuffer creates a new Buffer with the given data and pool reference
func NewBuffer(data []byte, pools *BytePool) *Buffer {
	buf := &Buffer{
		refCount: 1,
		pools:    pools,
	}
	buf.buf.Store(&data)
	return buf
}
