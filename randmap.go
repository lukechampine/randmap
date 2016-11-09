// Package randmap provides methods for accessing random elements of maps, and
// iterating through maps in random order.
package randmap

import (
	"unsafe"

	crand "crypto/rand"
	mrand "math/rand"
)

type emptyInterface struct {
	typ unsafe.Pointer
	val unsafe.Pointer
}

// mrand doesn't give us access to its globalRand, so we'll just use a
// function instead of an io.Reader
type randReader func(p []byte) (int, error)

func randInts(read randReader, mod uint8) (uintptr, uint8, uint8) {
	const ptrSize = unsafe.Sizeof(uintptr(0))
	var arena [ptrSize + 2]byte
	read(arena[:])
	uptr := *(*uintptr)(unsafe.Pointer(&arena[0]))
	return uptr, arena[ptrSize], arena[ptrSize+1] % mod
}

func randKey(m interface{}, src randReader) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	if h == nil || h.count == 0 {
		panic("empty map")
	}
	it := new(hiter)
	mod := maxOverflow(t, h) + 1
	r1, r2, ro := randInts(src, mod)
	for !randIter(t, h, it, r1, r2, ro) {
		r1, r2, ro = randInts(src, mod)
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.key),
		val: it.key,
	}))
}

func randVal(m interface{}, src randReader) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	if h == nil || h.count == 0 {
		panic("empty map")
	}
	it := new(hiter)
	mod := maxOverflow(t, h) + 1
	r1, r2, ro := randInts(src, mod)
	for !randIter(t, h, it, r1, r2, ro) {
		r1, r2, ro = randInts(src, mod)
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.elem),
		val: it.value,
	}))
}

// Key returns a uniform random key of m, which must be a non-empty map.
func Key(m interface{}) interface{} { return randKey(m, crand.Read) }

// Val returns a uniform random value of m, which must be a non-empty map.
func Val(m interface{}) interface{} { return randVal(m, crand.Read) }

// FastKey returns a pseudo-random key of m, which must be a non-empty map.
func FastKey(m interface{}) interface{} { return randKey(m, mrand.Read) }

// FastVal returns a pseudo-random value of m, which must be a non-empty map.
func FastVal(m interface{}) interface{} { return randVal(m, mrand.Read) }
