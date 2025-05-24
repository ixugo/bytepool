package bytepool

import "sync"

func NewPool[T any](f func() T) *Pool[T] {
	return &Pool[T]{p: sync.Pool{New: func() any { return f() }}}
}

type Pool[T any] struct {
	p sync.Pool
}

func (c *Pool[T]) Put(v T) {
	c.p.Put(v)
}

func (c *Pool[T]) Get() T {
	return c.p.Get().(T)
}
