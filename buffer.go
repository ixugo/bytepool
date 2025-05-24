package bytepool

import (
	"bytes"
	"sync/atomic"
)

type Buffer struct {
	buf      *bytes.Buffer
	refCount int32
	pools    *BytePool
}

func (b *Buffer) Bytes() ([]byte, func()) {
	b.Retain()
	bytes := b.buf.Bytes()
	return bytes, b.Release
}

func (b *Buffer) Release() {
	buf := b.buf
	if atomic.AddInt32(&b.refCount, -1) == 0 && buf != nil {
		b.pools.Put(buf.Bytes())
		b.buf = nil
	}
}

func (b *Buffer) Retain() {
	atomic.AddInt32(&b.refCount, 1)
}

func NewBuffer(data []byte, pools *BytePool) *Buffer {
	buf := Buffer{
		buf:      bytes.NewBuffer(data),
		refCount: 1,
		pools:    pools,
	}
	return &buf
}
