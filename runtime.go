package randmap

import "unsafe"

const (
	// Maximum number of key/value pairs a bucket can hold.
	bucketCntBits = 3
	bucketCnt     = 1 << bucketCntBits

	// data offset should be the size of the bmap struct, but needs to be
	// aligned correctly. For amd64p32 this means 64-bit alignment
	// even though pointers are 32 bit.
	dataOffset = unsafe.Offsetof(struct {
		b bmap
		v int64
	}{}.v)

	// Possible tophash values. We reserve a few possibilities for special marks.
	// Each bucket (including its overflow buckets, if any) will have either all or none of its
	// entries in the evacuated* states (except during the evacuate() method, which only happens
	// during map writes and thus no one else can observe the map during that time).
	empty          = 0 // cell is empty
	evacuatedEmpty = 1 // cell is empty, bucket is evacuated.
	evacuatedX     = 2 // key/value is valid.  Entry has been evacuated to first half of larger table.
	evacuatedY     = 3 // same as above, but evacuated to second half of larger table.
	minTopHash     = 4 // minimum tophash for a normal filled cell.

	noCheck = 1<<(8*unsafe.Sizeof(uintptr(0))) - 1

	kindNoPointers = 1 << 7
)

type (
	hmap struct {
		count int // # live cells == size of map.  Must be first (used by len() builtin)
		flags uint8
		B     uint8  // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
		hash0 uint32 // hash seed

		buckets    unsafe.Pointer // array of 2^B Buckets. may be nil if count==0.
		oldbuckets unsafe.Pointer // previous bucket array of half the size, non-nil only when growing
		nevacuate  uintptr        // progress counter for evacuation (buckets less than this have been evacuated)

		// If both key and value do not contain pointers and are inline, then we mark bucket
		// type as containing no pointers. This avoids scanning such maps.
		// However, bmap.overflow is a pointer. In order to keep overflow buckets
		// alive, we store pointers to all overflow buckets in hmap.overflow.
		// Overflow is used only if key and value do not contain pointers.
		// overflow[0] contains overflow buckets for hmap.buckets.
		// overflow[1] contains overflow buckets for hmap.oldbuckets.
		// The first indirection allows us to reduce static size of hmap.
		// The second indirection allows to store a pointer to the slice in hiter.
		overflow *[2]*[]*bmap
	}

	bmap struct {
		tophash [bucketCnt]uint8
		// Followed by bucketCnt keys and then bucketCnt values.
		// NOTE: packing all the keys together and then all the values together makes the
		// code a bit more complicated than alternating key/value/key/value/... but it allows
		// us to eliminate padding which would be needed for, e.g., map[int64]int8.
		// Followed by an overflow pointer.
	}

	hiter struct {
		key         unsafe.Pointer // Must be in first position.  Write nil to indicate iteration end (see cmd/internal/gc/range.go).
		value       unsafe.Pointer // Must be in second position (see cmd/internal/gc/range.go).
		t           *maptype
		h           *hmap
		buckets     unsafe.Pointer // bucket ptr at hash_iter initialization time
		bptr        *bmap          // current bucket
		overflow    [2]*[]*bmap    // keeps overflow buckets alive
		startBucket uintptr        // bucket iteration started at
		offset      uint8          // intra-bucket offset to start from during iteration (should be big enough to hold bucketCnt-1)
		wrapped     bool           // already wrapped around from end of bucket array to beginning
		B           uint8
		i           uint8
		bucket      uintptr
		checkBucket uintptr
	}

	_type struct {
		size       uintptr
		ptrdata    uintptr // size of memory prefix holding all pointers
		hash       uint32
		tflag      uint8
		align      uint8
		fieldalign uint8
		kind       uint8
		alg        *typeAlg
		// gcdata stores the GC type data for the garbage collector.
		// If the KindGCProg bit is set in kind, gcdata is a GC program.
		// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
		gcdata    uintptr
		str       int32
		ptrToThis int32
	}

	maptype struct {
		typ           _type
		key           *_type
		elem          *_type
		bucket        *_type // internal type representing a hash bucket
		hmap          *_type // internal type representing a hmap
		keysize       uint8  // size of key slot
		indirectkey   bool   // store ptr to key instead of key itself
		valuesize     uint8  // size of value slot
		indirectvalue bool   // store ptr to value instead of value itself
		bucketsize    uint16 // size of bucket
		reflexivekey  bool   // true if k==k for all keys
		needkeyupdate bool   // true if we need to update key on an overwrite
	}

	// typeAlg is also copied/used in reflect/type.go.
	// keep them in sync.
	typeAlg struct {
		// function for hashing objects of this type
		// (ptr to object, seed) -> hash
		hash func(unsafe.Pointer, uintptr) uintptr
		// function for comparing objects of this type
		// (ptr to object A, ptr to object B) -> ==?
		equal func(unsafe.Pointer, unsafe.Pointer) bool
	}
)

func evacuated(b *bmap) bool {
	h := b.tophash[0]
	return h > empty && h < minTopHash
}

func (b *bmap) overflow(t *maptype) *bmap {
	return *(**bmap)(add(unsafe.Pointer(b), uintptr(t.bucketsize)-unsafe.Sizeof(uintptr(0))))
}

func (h *hmap) setoverflow(t *maptype, b, ovf *bmap) {
	if t.bucket.kind&kindNoPointers != 0 {
		h.createOverflow()
		*h.overflow[0] = append(*h.overflow[0], ovf)
	}
	*(**bmap)(add(unsafe.Pointer(b), uintptr(t.bucketsize)-unsafe.Sizeof(uintptr(0)))) = ovf
}

func (h *hmap) createOverflow() {
	if h.overflow == nil {
		h.overflow = new([2]*[]*bmap)
	}
	if h.overflow[0] == nil {
		h.overflow[0] = new([]*bmap)
	}
}

func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

func maxOverflow(t *maptype, h *hmap) uint8 {
	var buckets unsafe.Pointer
	if h.oldbuckets != nil && !evacuated((*bmap)(h.oldbuckets)) {
		buckets = h.oldbuckets
	} else {
		buckets = h.buckets
	}

	var max uint8
	for i := uintptr(0); i < (1 << h.B); i++ {
		var over uint8
		for b := (*bmap)(add(buckets, i*uintptr(t.bucketsize))).overflow(t); b != nil; over++ {
			b = b.overflow(t)
		}
		if over > max {
			max = over
		}
	}
	return max
}

// randIter moves it to a random index in hmap, which may or may not contain
// valid data. It returns true if the data is valid, and false otherwise.
func randIter(t *maptype, h *hmap, it *hiter, r1 uintptr, r2, ro uint8) bool {
	// Clear pointer fields so garbage collector does not complain.
	it.key = nil
	it.value = nil
	it.t = nil
	it.h = nil
	it.buckets = nil
	it.bptr = nil
	it.overflow[0] = nil
	it.overflow[1] = nil

	it.t = t
	it.h = h

	// grab snapshot of bucket state
	it.B = h.B
	it.buckets = h.buckets
	if t.bucket.kind&kindNoPointers != 0 {
		// Allocate the current slice and remember pointers to both current and old.
		// This preserves all relevant overflow buckets alive even if
		// the table grows and/or overflow buckets are added to the table
		// while we are iterating.
		h.createOverflow()
		it.overflow = *h.overflow
	}

	// decide where to start
	// NOTE: we can safely use & (instead of the usual modulus) because the
	// masks are powers of two
	bucket := r1 & (uintptr(1)<<h.B - 1)
	offi := r2 & (bucketCnt - 1)

	// mapiternext

	var b *bmap
	checkBucket := it.checkBucket
	alg := t.key.alg

	if h.oldbuckets != nil && it.B == h.B {
		// Iterator was started in the middle of a grow, and the grow isn't done yet.
		// If the bucket we're looking at hasn't been filled in yet (i.e. the old
		// bucket hasn't been evacuated) then we need to iterate through the old
		// bucket and only return the ones that will be migrated to this bucket.
		oldbucket := bucket & (uintptr(1)<<(it.B-1) - 1)
		b = (*bmap)(add(h.oldbuckets, oldbucket*uintptr(t.bucketsize)))
		if !evacuated(b) {
			checkBucket = bucket
		} else {
			b = (*bmap)(add(it.buckets, bucket*uintptr(t.bucketsize)))
			checkBucket = noCheck
		}
	} else {
		b = (*bmap)(add(it.buckets, bucket*uintptr(t.bucketsize)))
		checkBucket = noCheck
	}

	// select an overflow bucket
	for i := uint8(0); i < ro; i++ {
		b = b.overflow(t)
		if b == nil {
			return false
		}
	}

	k := add(unsafe.Pointer(b), dataOffset+uintptr(offi)*uintptr(t.keysize))
	v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+uintptr(offi)*uintptr(t.valuesize))
	if b.tophash[offi] == empty || b.tophash[offi] == evacuatedEmpty {
		// bucket is empty
		return false
	}
	if checkBucket != noCheck {
		// Special case: iterator was started during a grow and the
		// grow is not done yet. We're working on a bucket whose
		// oldbucket has not been evacuated yet. Or at least, it wasn't
		// evacuated when we started the bucket. So we're iterating
		// through the oldbucket, skipping any keys that will go
		// to the other new bucket (each oldbucket expands to two
		// buckets during a grow).
		k2 := k
		if t.indirectkey {
			k2 = *((*unsafe.Pointer)(k2))
		}
		if t.reflexivekey || alg.equal(k2, k2) {
			// If the item in the oldbucket is not destined for
			// the current new bucket in the iteration, skip it.
			hash := alg.hash(k2, uintptr(h.hash0))
			if hash&(uintptr(1)<<it.B-1) != checkBucket {
				return false
			}
		} else {
			// Hash isn't repeatable if k != k (NaNs).  We need a
			// repeatable and randomish choice of which direction
			// to send NaNs during evacuation. We'll use the low
			// bit of tophash to decide which way NaNs go.
			// NOTE: this case is why we need two evacuate tophash
			// values, evacuatedX and evacuatedY, that differ in
			// their low bit.
			if checkBucket>>(it.B-1) != uintptr(b.tophash[offi]&1) {
				return false
			}
		}
	}
	if b.tophash[offi] != evacuatedX && b.tophash[offi] != evacuatedY {
		// this is the golden data, we can return it.
		if t.indirectkey {
			k = *((*unsafe.Pointer)(k))
		}
		it.key = k
		if t.indirectvalue {
			v = *((*unsafe.Pointer)(v))
		}
		it.value = v
	} else {
		// The hash table has grown since the iterator was started.
		// The golden data for this key is now somewhere else.
		k2 := k
		if t.indirectkey {
			k2 = *((*unsafe.Pointer)(k2))
		}
		if t.reflexivekey || alg.equal(k2, k2) {
			// Check the current hash table for the data.
			// This code handles the case where the key
			// has been deleted, updated, or deleted and reinserted.
			// NOTE: we need to regrab the key as it has potentially been
			// updated to an equal() but not identical key (e.g. +0.0 vs -0.0).
			rk, rv := mapaccessK(t, h, k2)
			if rk == nil {
				return false // key has been deleted
			}
			it.key = rk
			it.value = rv
		} else {
			// if key!=key then the entry can't be deleted or
			// updated, so we can just return it. That's lucky for
			// us because when key!=key we can't look it up
			// successfully in the current table.
			it.key = k2
			if t.indirectvalue {
				v = *((*unsafe.Pointer)(v))
			}
			it.value = v
		}
	}

	return true
}

// returns both key and value. Used by map iterator
func mapaccessK(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, unsafe.Pointer) {
	if h == nil || h.count == 0 {
		return nil, nil
	}
	alg := t.key.alg
	hash := alg.hash(key, uintptr(h.hash0))
	m := uintptr(1)<<h.B - 1
	b := (*bmap)(unsafe.Pointer(uintptr(h.buckets) + (hash&m)*uintptr(t.bucketsize)))
	if c := h.oldbuckets; c != nil {
		oldb := (*bmap)(unsafe.Pointer(uintptr(c) + (hash&(m>>1))*uintptr(t.bucketsize)))
		if !evacuated(oldb) {
			b = oldb
		}
	}
	top := uint8(hash >> (unsafe.Sizeof(uintptr(0))*8 - 8))
	if top < minTopHash {
		top += minTopHash
	}
	for {
		for i := uintptr(0); i < bucketCnt; i++ {
			if b.tophash[i] != top {
				continue
			}
			k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
			if t.indirectkey {
				k = *((*unsafe.Pointer)(k))
			}
			if alg.equal(key, k) {
				v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
				if t.indirectvalue {
					v = *((*unsafe.Pointer)(v))
				}
				return k, v
			}
		}
		b = b.overflow(t)
		if b == nil {
			return nil, nil
		}
	}
}
