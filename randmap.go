// Package randmap provides methods for accessing random elements of maps, and
// iterating through maps in random order.
package randmap

import (
	"reflect"
	"unsafe"

	crand "crypto/rand"
	mrand "math/rand"

	"github.com/lukechampine/randmap/perm"
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

// An Iterator iterates over a map in random or pseudorandom order. It is
// intended to be used in a for loop like so:
//
//  m := make(map[int]int)
//  var k, v int
//  i := Iterator(m, &k, &v)
//  for i.Next() {
//      // use k and v
//  }
//
type Iterator struct {
	// permutation generator
	gen interface {
		Next() (uint32, bool)
	}

	it   *hiter
	k, v reflect.Value

	// constants
	t    *maptype
	h    *hmap
	over uint32
}

// Next advances the Iterator to the next element in the map, storing its key
// and value in the pointers passed during initialization. It returns false
// when all of the elements have been enumerated.
func (i *Iterator) Next() bool {
	if i == nil {
		return false
	}
	t, h, it := i.t, i.h, i.it

	for {
		r, ok := i.gen.Next()
		if !ok {
			return false
		}

		bucket := uintptr(r / (i.over * bucketCnt))
		over := (r / bucketCnt) % i.over
		offi := r % bucketCnt
		if mapaccessi(t, h, it, bucket, uint8(over), uint8(offi)) {
			// unfortunately, there doesn't seem to be a faster way than this
			k := *(*interface{})(unsafe.Pointer(&emptyInterface{
				typ: unsafe.Pointer(t.key),
				val: it.key,
			}))
			v := *(*interface{})(unsafe.Pointer(&emptyInterface{
				typ: unsafe.Pointer(t.elem),
				val: it.value,
			}))
			i.k.Set(reflect.ValueOf(k))
			i.v.Set(reflect.ValueOf(v))
			return true
		}
	}
}

func randIter(m, k, v interface{}, read randReader) *Iterator {
	mt, kt, vt := reflect.TypeOf(m), reflect.TypeOf(k), reflect.TypeOf(v)
	if exp := reflect.PtrTo(mt.Key()); kt != exp {
		panic("wrong type for k: expected " + exp.String() + ", got " + kt.String())
	} else if exp = reflect.PtrTo(mt.Elem()); vt != exp {
		panic("wrong type for v: expected " + exp.String() + ", got " + vt.String())
	}

	// determine total rand space for m
	ei := (*emptyInterface)(unsafe.Pointer(&m))
	t := (*maptype)(ei.typ)
	h := (*hmap)(ei.val)
	if h == nil || h.count == 0 {
		return nil
	}
	numOver := uint32(maxOverflow(t, h) + 1)
	numBuckets := uint32(1 << h.B)
	space := numBuckets * numOver * bucketCnt

	// create a permutation generator for the space
	var seed [4]byte
	read(seed[:])
	g := perm.NewGenerator(space, *(*uint32)(unsafe.Pointer(&seed[0])))

	// grab pointers to k and v's memory
	kptr := reflect.ValueOf(k).Elem()
	vptr := reflect.ValueOf(v).Elem()

	return &Iterator{
		gen:  g,
		it:   new(hiter),
		k:    kptr,
		v:    vptr,
		t:    t,
		h:    h,
		over: numOver,
	}
}

// Key returns a uniform random key of m, which must be a non-empty map.
func Key(m interface{}) interface{} { return randKey(m, crand.Read) }

// Val returns a uniform random value of m, which must be a non-empty map.
func Val(m interface{}) interface{} { return randVal(m, crand.Read) }

// Iter returns a random iterator for m. Each call to Next will store the next
// key/value pair in k and v, which must be pointers. Modifying the map during
// iteration will result in undefined behavior.
func Iter(m, k, v interface{}) *Iterator { return randIter(m, k, v, crand.Read) }

// FastKey returns a pseudorandom key of m, which must be a non-empty map.
func FastKey(m interface{}) interface{} { return randKey(m, mrand.Read) }

// FastVal returns a pseudorandom value of m, which must be a non-empty map.
func FastVal(m interface{}) interface{} { return randVal(m, mrand.Read) }

// FastIter returns a pseudorandom iterator for m. Each call to Next will
// store the next key/value pair in k and v, which must be pointers. Modifying
// the map during iteration will result in undefined behavior.
func FastIter(m, k, v interface{}) *Iterator { return randIter(m, k, v, mrand.Read) }
