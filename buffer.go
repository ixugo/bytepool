package bytepool

import (
	"sync/atomic"
)

type Buffer struct {
	buf      []byte
	refCount int32
	pools    *BytePool
}

func (b *Buffer) Bytes() ([]byte, func()) {
	b.Retain()
	return b.buf, b.Release
}

func (b *Buffer) Release() {
	buf := b.buf
	if atomic.AddInt32(&b.refCount, -1) == 0 && buf != nil {
		b.pools.Put(buf)
		b.buf = nil
	}
}

func (b *Buffer) Retain() {
	atomic.AddInt32(&b.refCount, 1)
}

func NewBuffer(data []byte, pools *BytePool) *Buffer {
	buf := Buffer{
		buf:      data,
		refCount: 1,
		pools:    pools,
	}
	return &buf
}
