package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type testObject struct {
	value      string
	resetCalls int
}

func (o *testObject) Reset() {
	if o == nil {
		return
	}
	o.value = ""
	o.resetCalls++
}

func TestNewGetPut_ResetCalledAndObjectReused(t *testing.T) {
	p, err := New(func() *testObject {
		return &testObject{value: "new"}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := p.Get()
	obj.value = "dirty"

	p.Put(obj)

	got := p.Get()
	if got != obj {
		t.Fatal("expected object reuse after Put/Get")
	}
	if got.value != "" {
		t.Fatalf("expected reset value, got %q", got.value)
	}
	if got.resetCalls != 1 {
		t.Fatalf("expected one reset call, got %d", got.resetCalls)
	}
}

func TestGet_UsesConstructorWhenPoolEmpty(t *testing.T) {
	var created int
	p, err := New(func() *testObject {
		created++
		return &testObject{}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = p.Get()
	_ = p.Get()

	if created != 2 {
		t.Fatalf("expected constructor to be called twice, got %d", created)
	}
}

func TestPool_ConcurrentGetPut(t *testing.T) {
	p, err := New(func() *testObject {
		return &testObject{value: "new"}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const goroutines = 16
	const iterations = 200

	var wg sync.WaitGroup
	var totalResetCalls atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				obj := p.Get()
				obj.value = "dirty"
				p.Put(obj)
				totalResetCalls.Add(1)
			}
		}()
	}
	wg.Wait()

	if totalResetCalls.Load() != goroutines*iterations {
		t.Fatalf("unexpected total reset count marker: got %d", totalResetCalls.Load())
	}
}

func TestNew_ReturnsErrorOnNilConstructor(t *testing.T) {
	p, err := New[*testObject](nil)
	if p != nil {
		t.Fatal("expected nil pool")
	}
	if !errors.Is(err, ErrNilNewFn) {
		t.Fatalf("expected ErrNilNewFn, got %v", err)
	}
}
