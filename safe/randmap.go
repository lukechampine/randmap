package randmap

import (
	"math/big"
	"reflect"

	crand "crypto/rand"
	mrand "math/rand"
)

// A randIntn function returns a random value in [0, n).
type randIntn func(n int) int

func cRandInt(n int) int {
	i, _ := crand.Int(crand.Reader, big.NewInt(int64(n)))
	return int(i.Int64())
}

func mRandInt(n int) int {
	return mrand.Intn(n)
}

func randKey(m interface{}, Intn randIntn) interface{} {
	mv := reflect.ValueOf(m)
	keys := mv.MapKeys()
	return keys[Intn(len(keys))].Interface()
}

func randVal(m interface{}, Intn randIntn) interface{} {
	mv := reflect.ValueOf(m)
	keys := mv.MapKeys()
	val := mv.MapIndex(keys[Intn(len(keys))])
	return val.Interface()
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
	// map, stored so that we can lookup values via keys
	m reflect.Value
	// random permutation of map keys, resliced each time we iterate
	perm []reflect.Value
	// settable Values for k and v
	k, v reflect.Value
}

// Next advances the Iterator to the next element in the map, storing its key
// and value in the pointers passed during initialization. It returns false
// when all of the elements have been enumerated.
func (i *Iterator) Next() bool {
	if i == nil || len(i.perm) == 0 {
		return false
	}
	var k reflect.Value
	k, i.perm = i.perm[0], i.perm[1:]
	i.k.Set(k)
	i.v.Set(i.m.MapIndex(k))
	return true
}

func randIter(m, k, v interface{}, Intn randIntn) *Iterator {
	mt, kt, vt := reflect.TypeOf(m), reflect.TypeOf(k), reflect.TypeOf(v)
	if exp := reflect.PtrTo(mt.Key()); kt != exp {
		panic("wrong type for k: expected " + exp.String() + ", got " + kt.String())
	} else if exp = reflect.PtrTo(mt.Elem()); vt != exp {
		panic("wrong type for v: expected " + exp.String() + ", got " + vt.String())
	}

	// grab pointers to k and v's memory
	kptr := reflect.ValueOf(k).Elem()
	vptr := reflect.ValueOf(v).Elem()

	// create a random permutation of m's keys
	mv := reflect.ValueOf(m)
	keys := mv.MapKeys()
	for i := len(keys) - 1; i >= 1; i-- {
		j := Intn(i + 1)
		keys[i], keys[j] = keys[j], keys[i]
	}

	return &Iterator{
		m:    mv,
		perm: keys,
		k:    kptr,
		v:    vptr,
	}
}

// Key returns a uniform random key of m, which must be a non-empty map.
func Key(m interface{}) interface{} { return randKey(m, cRandInt) }

// Val returns a uniform random value of m, which must be a non-empty map.
func Val(m interface{}) interface{} { return randVal(m, cRandInt) }

// Iter returns a random iterator for m. Each call to Next will store the next
// key/value pair in k and v, which must be pointers. Modifying the map during
// iteration will result in undefined behavior.
func Iter(m, k, v interface{}) *Iterator { return randIter(m, k, v, cRandInt) }

// FastKey returns a pseudorandom key of m, which must be a non-empty map.
func FastKey(m interface{}) interface{} { return randKey(m, mRandInt) }

// FastVal returns a pseudorandom value of m, which must be a non-empty map.
func FastVal(m interface{}) interface{} { return randVal(m, mRandInt) }

// FastIter returns a pseudorandom iterator for m. Each call to Next will
// store the next key/value pair in k and v, which must be pointers. Modifying
// the map during iteration will result in undefined behavior.
func FastIter(m, k, v interface{}) *Iterator { return randIter(m, k, v, mRandInt) }
