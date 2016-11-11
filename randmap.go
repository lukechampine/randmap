// Package randmap provides methods for accessing random elements of maps, and
// iterating through maps in random order.
package randmap

import (
	"reflect"
	"unsafe"

	crand "crypto/rand"
	mrand "math/rand"
)

const ptrSize = unsafe.Sizeof(uintptr(0))

type emptyInterface struct {
	typ unsafe.Pointer
	val unsafe.Pointer
}

// mrand doesn't give us access to its globalRand, so we'll just use a
// function instead of an io.Reader
type randReader func(p []byte) (int, error)

func randInts(read randReader, numBuckets uintptr, numOver uint8) (uintptr, uint8, uint8) {
	space := numBuckets * uintptr(numOver) * bucketCnt
	var arena [ptrSize]byte
	read(arena[:])
	r := *(*uintptr)(unsafe.Pointer(&arena[0])) % space

	bucket := r / (uintptr(numOver) * bucketCnt)
	over := (r / bucketCnt) % uintptr(numOver)
	offi := r % bucketCnt

	return bucket, uint8(over), uint8(offi)
}

func randKey(m interface{}, src randReader) interface{} {
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	if h == nil || h.count == 0 {
		panic("empty map")
	}
	it := new(hiter)
	numBuckets := uintptr(1) << h.B
	numOver := maxOverflow(t, h) + 1
	bucket, over, offi := randInts(src, numBuckets, numOver)
	for !mapaccessi(t, h, it, bucket, over, offi) {
		bucket, over, offi = randInts(src, numBuckets, numOver)
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
	numBuckets := uintptr(1) << h.B
	numOver := maxOverflow(t, h) + 1
	bucket, over, offi := randInts(src, numBuckets, numOver)
	for !mapaccessi(t, h, it, bucket, over, offi) {
		bucket, over, offi = randInts(src, numBuckets, numOver)
	}
	return *(*interface{})(unsafe.Pointer(&emptyInterface{
		typ: unsafe.Pointer(t.elem),
		val: it.value,
	}))
}

func randIter(m, fn interface{}, read randReader) {
	mt := reflect.TypeOf(m)
	fv, ft := reflect.ValueOf(fn), reflect.TypeOf(fn)
	if ft.Kind() != reflect.Func || ft.NumIn() != 2 || ft.In(0) != mt.Key() || ft.In(1) != mt.Elem() {
		exp := reflect.FuncOf([]reflect.Type{mt.Key(), mt.Elem()}, nil, false)
		panic("wrong type for fn: expected " + exp.String() + ", got " + ft.String())
	}

	// determine total rand space for m
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	if h == nil || h.count == 0 {
		return
	}
	numOver := uint32(maxOverflow(t, h) + 1)
	numBuckets := uint32(1 << h.B)
	space := numBuckets * numOver * bucketCnt

	// create a permutation generator for the space
	var seed [4]byte
	read(seed[:])
	g := newGenerator(space, *(*uint32)(unsafe.Pointer(&seed[0])))

	// iterate through the permutation, accessing each cell and calling fn on
	// the non-empty ones
	it := new(hiter)
	fnIns := make([]reflect.Value, 2)
	g.Iter(func(r uint32) {
		bucket := uintptr(r / (numOver * bucketCnt))
		over := (r / bucketCnt) % numOver
		offi := r % bucketCnt
		if mapaccessi(t, h, it, bucket, uint8(over), uint8(offi)) {
			k := *(*interface{})(unsafe.Pointer(&emptyInterface{
				typ: unsafe.Pointer(t.key),
				val: it.key,
			}))
			v := *(*interface{})(unsafe.Pointer(&emptyInterface{
				typ: unsafe.Pointer(t.elem),
				val: it.value,
			}))
			fnIns[0] = reflect.ValueOf(k)
			fnIns[1] = reflect.ValueOf(v)
			fv.Call(fnIns)
		}
	})
}

// Key returns a uniform random key of m, which must be a non-empty map.
func Key(m interface{}) interface{} { return randKey(m, crand.Read) }

// Val returns a uniform random value of m, which must be a non-empty map.
func Val(m interface{}) interface{} { return randVal(m, crand.Read) }

// Iter calls fn on the key/value pairs of m in random order. fn must be a
// function of two arguments whose types match the map's key and value types.
// Return values of fn are discarded.
func Iter(m, fn interface{}) { randIter(m, fn, crand.Read) }

// FastKey returns a pseudo-random key of m, which must be a non-empty map.
func FastKey(m interface{}) interface{} { return randKey(m, mrand.Read) }

// FastVal returns a pseudo-random value of m, which must be a non-empty map.
func FastVal(m interface{}) interface{} { return randVal(m, mrand.Read) }

// FastIter calls fn on the key/value pairs of m in pseudo-random order. fn
// must be a function of two arguments whose types match the map's key and
// value types. Return values of fn are discarded.
func FastIter(m, fn interface{}) { randIter(m, fn, mrand.Read) }
