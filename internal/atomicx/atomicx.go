// Package atomicx forwards compatibility with go1.19
package atomicx

import (
	"sync"
	"sync/atomic"
)

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// A Bool is an atomic boolean value.
// The zero value is false.
type Bool struct {
	_ noCopy
	v uint32
}

// Load atomically loads and returns the value stored in x.
func (x *Bool) Load() bool { return atomic.LoadUint32(&x.v) != 0 }

// Store atomically stores val into x.
func (x *Bool) Store(val bool) { atomic.StoreUint32(&x.v, b32(val)) }

// Swap atomically stores new into x and returns the previous value.
func (x *Bool) Swap(new bool) (old bool) { return atomic.SwapUint32(&x.v, b32(new)) != 0 }

// CompareAndSwap executes the compare-and-swap operation for the boolean value x.
func (x *Bool) CompareAndSwap(old, new bool) (swapped bool) {
	return atomic.CompareAndSwapUint32(&x.v, b32(old), b32(new))
}

// b32 returns a uint32 0 or 1 representing b.
func b32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func NewValue[T comparable](v T) V[T] {
	return V[T]{_v: &_v[T]{v: v}}
}

type _v[T comparable] struct {
	sync.RWMutex
	v T
}

type V[T comparable] struct {
	*_v[T]
}

// Load atomically loads and returns the value stored in x.
func (t *V[T]) Load() T {
	t.RWMutex.RLock()
	defer t.RWMutex.RUnlock()
	return t.v
}

// Alias for Swap.
func (t *V[T]) Store(v T) (old T) {
	return t.Swap(v)
}

// Swap atomically stores new into x and returns the previous value.
func (t *V[T]) Swap(v T) (old T) {
	t.RWMutex.Lock()
	defer t.RWMutex.Unlock()
	old, t.v = t.v, v
	return old
}

// Swap atomically stores new into x and returns the previous value.
func (t *V[T]) unsafeSwap(v T) (old T) {
	old, t.v = t.v, v
	return old
}

// CompareAndSwap executes the compare-and-swap operation for the boolean value x.
func (t *V[T]) CompareAndSwap(old, new T) (swapped bool) {
	t.RWMutex.Lock()
	defer t.RWMutex.Unlock()
	if t.v != old {
		return false
	}

	t.unsafeSwap(new)
	return true
}
