package bytepool

import "sync"

// NewPool creates a new generic pool with the given factory function
func NewPool[T any](f func() T) *Pool[T] {
	return &Pool[T]{p: sync.Pool{New: func() any { return f() }}}
}

// Pool is a generic wrapper around sync.Pool
type Pool[T any] struct {
	p sync.Pool
}

// Put adds an item to the pool
func (c *Pool[T]) Put(v T) {
	c.p.Put(v)
}

// Get retrieves an item from the pool
func (c *Pool[T]) Get() T {
	return c.p.Get().(T)
}
