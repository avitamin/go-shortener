package pool

import "sync"

// Resettable описывает типы, которые умеют сбрасывать своё состояние.
type Resettable interface {
	Reset()
}

// Pool — generic-обёртка над sync.Pool для объектов с методом Reset.
type Pool[T Resettable] struct {
	p sync.Pool
}

// New создает пул объектов типа T.
func New[T Resettable](newFn func() T) *Pool[T] {
	if newFn == nil {
		panic("pool: new function is nil")
	}

	return &Pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
