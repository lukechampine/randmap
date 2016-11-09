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

type randFn func(mod int) (uintptr, uintptr, uintptr)

// the global mrand functions are guarded by a mutex. To avoid unnecessary
// locking, we use our own Rand.
var urand = mrand.New(mrand.NewSource(1))

func pseudoUint32s(mod int) (uintptr, uintptr, uintptr) {
	i64 := urand.Int63()
	return uintptr(i64), uintptr(i64 >> 32), uintptr(urand.Intn(mod))
}

var max = new(big.Int).SetUint64(^uint64(0))

func cryptoUint32s(mod int) (uintptr, uintptr, uintptr) {
	r, _ := crand.Int(crand.Reader, max)
	r2, _ := crand.Int(crand.Reader, big.NewInt(int64(mod)))
	u64 := r.Uint64()
	return uintptr(u64), uintptr(u64 >> 32), uintptr(r2.Uint64())
}

func randKey(m interface{}, rand randFn) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	mod := maxOverflow(t, h) + 1
	r1, r2, ro := rand(mod)
	for !mapiterinit(t, h, it, r1, r2, ro) {
		r1, r2, ro = rand(mod)
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.key),
		val: it.key,
	}))
}

func randVal(m interface{}, rand randFn) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	mod := maxOverflow(t, h) + 1
	r1, r2, ro := rand(mod)
	for !mapiterinit(t, h, it, r1, r2, ro) {
		r1, r2, ro = rand(mod)
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
