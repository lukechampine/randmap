// Package randmap provides methods for accessing random elements of maps, and
// iterating through maps in random order.
package randmap

import (
	"math/big"
	"unsafe"

	crand "crypto/rand"
	mrand "math/rand"
)

type emptyInterface struct {
	typ unsafe.Pointer
	val unsafe.Pointer
}

// the global mrand functions are guarded by a mutex. To avoid unnecessary
// locking, we use our own Rand.
var urand = mrand.New(mrand.NewSource(1))

func pseudoUint32s() (uint32, uint32) {
	i64 := urand.Int63()
	return uint32(i64), uint32(i64 >> 32)
}

var max = new(big.Int).SetUint64(^uint64(0))

func cryptoUint32s() (uint32, uint32) {
	r, _ := crand.Int(crand.Reader, max)
	u64 := r.Uint64()
	return uint32(u64), uint32(u64 >> 32)
}

func randKey(m interface{}, randFn func() (uint32, uint32)) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	r1, r2 := randFn()
	for !mapiterinit(t, h, it, uintptr(r1), uintptr(r2)) {
		r1, r2 = randFn()
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.key),
		val: it.key,
	}))
}

func randVal(m interface{}, randFn func() (uint32, uint32)) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	r1, r2 := randFn()
	for !mapiterinit(t, h, it, uintptr(r1), uintptr(r2)) {
		r1, r2 = randFn()
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.elem),
		val: it.value,
	}))
}

// Key returns a uniform random key of m, which must be a non-empty map.
func Key(m interface{}) interface{} { return randKey(m, cryptoUint32s) }

// Val returns a uniform random value of m, which must be a non-empty map.
func Val(m interface{}) interface{} { return randVal(m, cryptoUint32s) }

// FastKey returns a pseudo-random key of m, which must be a non-empty map.
func FastKey(m interface{}) interface{} { return randKey(m, pseudoUint32s) }

// FastVal returns a pseudo-random value of m, which must be a non-empty map.
func FastVal(m interface{}) interface{} { return randVal(m, pseudoUint32s) }
