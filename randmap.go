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

var max = new(big.Int).SetUint64(uint64(^uint32(0)))

func csrUintptr() uintptr {
	r, _ := crand.Int(crand.Reader, max)
	return uintptr(r.Uint64())
}

func iter(m interface{}, r1, r2 uintptr) *hiter {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	it := new(hiter)
	mapiterinit(t, h, it, r1, r2)
	return it
}

func Key(m interface{}) interface{} {
	it := iter(m, csrUintptr(), csrUintptr())
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.key),
		val: it.key,
	}))
}

func Val(m interface{}) interface{} {
	it := iter(m, csrUintptr(), csrUintptr())
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.elem),
		val: it.value,
	}))
}

func FastKey(m interface{}) interface{} {
	it := iter(m, uintptr(mrand.Uint32()), uintptr(mrand.Uint32()))
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.key),
		val: it.key,
	}))
}

func FastVal(m interface{}) interface{} {
	it := iter(m, uintptr(mrand.Uint32()), uintptr(mrand.Uint32()))
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(it.t.elem),
		val: it.value,
	}))
}
