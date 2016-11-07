package randmap

import (
	"math/big"
	"unsafe"

	crand "crypto/rand"
	mrand "math/rand"
)

// the global mrand functions are guarded by a mutex. To avoid unnecessary
// locking, we use our own Rand.
var urand = mrand.New(mrand.NewSource(1))

type emptyInterface struct {
	typ unsafe.Pointer
	val unsafe.Pointer
}

var max = new(big.Int).SetUint64(uint64(^uint32(0)))

func csrUint32s() (uint32, uint32) {
	r, _ := crand.Int(crand.Reader, max)
	u64 := r.Uint64()
	return uint32(u64), uint32(u64 >> 32)
}

func Key(m interface{}) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	r1, r2 := csrUint32s()
	for !mapiterinit(t, h, it, uintptr(r1), uintptr(r2)) {
		r1, r2 = csrUint32s()
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.key),
		val: it.key,
	}))
}

func Val(m interface{}) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	r1, r2 := csrUint32s()
	for !mapiterinit(t, h, it, uintptr(r1), uintptr(r2)) {
		r1, r2 = csrUint32s()
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.elem),
		val: it.value,
	}))
}

func FastKey(m interface{}) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	for !mapiterinit(t, h, it, uintptr(urand.Uint32()), uintptr(urand.Uint32())) {
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.key),
		val: it.key,
	}))
}

func FastVal(m interface{}) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	for !mapiterinit(t, h, it, uintptr(urand.Uint32()), uintptr(urand.Uint32())) {
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.elem),
		val: it.value,
	}))
}
