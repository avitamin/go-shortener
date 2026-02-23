package pool

import (
	"errors"
	"sync"
)

// Resettable описывает типы, которые умеют сбрасывать своё состояние.
type Resettable interface {
	Reset()
}

// Pool — generic-обёртка над sync.Pool для объектов с методом Reset.
type Pool[T Resettable] struct {
	p sync.Pool
}

var ErrNilNewFn = errors.New("pool: new function is nil")

// New создает пул объектов типа T.
func New[T Resettable](newFn func() T) (*Pool[T], error) {
	if newFn == nil {
		return nil, ErrNilNewFn
	}

	return &Pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}, nil
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
