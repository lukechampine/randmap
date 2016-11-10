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
		key      unsafe.Pointer // Must be in first position.  Write nil to indicate iteration end (see cmd/internal/gc/range.go).
		value    unsafe.Pointer // Must be in second position (see cmd/internal/gc/range.go).
		overflow [2]*[]*bmap    // keeps overflow buckets alive
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

// maxOverflow returns the length of the longest bucket chain in the map.
func maxOverflow(t *maptype, h *hmap) uint8 {
	var max uint8
	if h.oldbuckets != nil {
		for i := uintptr(0); i < (1 << (h.B - 1)); i++ {
			var over uint8
			b := (*bmap)(add(h.oldbuckets, i*uintptr(t.bucketsize)))
			if evacuated(b) {
				continue
			}
			for b = b.overflow(t); b != nil; over++ {
				b = b.overflow(t)
			}
			if over > max {
				max = over
			}
		}
	}
	for i := uintptr(0); i < (1 << h.B); i++ {
		var over uint8
		for b := (*bmap)(add(h.buckets, i*uintptr(t.bucketsize))).overflow(t); b != nil; over++ {
			b = b.overflow(t)
		}
		if over > max {
			max = over
		}
	}
	return max
}

// randIter moves 'it' to a random index in hmap, which may or may not contain
// valid data. It returns true if the data is valid, and false otherwise.
func randIter(t *maptype, h *hmap, it *hiter, r1 uintptr, r2, ro uint8) bool {
	// grab snapshot of bucket state
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
	b := (*bmap)(add(h.buckets, bucket*uintptr(t.bucketsize)))

	checkBucket := false
	if h.oldbuckets != nil {
		// Iterator was started in the middle of a grow, and the grow isn't done yet.
		// If the bucket we're looking at hasn't been filled in yet (i.e. the old
		// bucket hasn't been evacuated) then we need to use that pointer instead.
		oldbucket := bucket & (uintptr(1)<<(h.B-1) - 1)
		oldB := (*bmap)(add(h.oldbuckets, oldbucket*uintptr(t.bucketsize)))
		if !evacuated(oldB) {
			b = oldB
			checkBucket = true
		}
	}

	// select a random overflow bucket
	for i := uint8(0); i < ro; i++ {
		b = b.overflow(t)
		if b == nil {
			return false
		}
	}

	// check that bucket is not empty
	if b.tophash[offi] == empty || b.tophash[offi] == evacuatedEmpty {
		return false
	}

	// grab the key and value
	k := add(unsafe.Pointer(b), dataOffset+uintptr(offi)*uintptr(t.keysize))
	v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+uintptr(offi)*uintptr(t.valuesize))
	if t.indirectkey {
		k = *((*unsafe.Pointer)(k))
	}
	if t.indirectvalue {
		v = *((*unsafe.Pointer)(v))
	}

	// if this is an old bucket, we need to check whether this key is destined
	// for the new bucket. Otherwise, we will have a 2x bias towards oldbucket
	// values, since two different bucket selections can result in the same
	// oldbucket.
	if checkBucket {
		if t.reflexivekey || t.key.alg.equal(k, k) {
			// If the item in the oldbucket is not destined for
			// the current new bucket in the iteration, skip it.
			hash := t.key.alg.hash(k, uintptr(h.hash0))
			if hash&(uintptr(1)<<h.B-1) != bucket {
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
			if bucket>>(h.B-1) != uintptr(b.tophash[offi]&1) {
				return false
			}
		}
	}

	it.key = k
	it.value = v
	return true
}

// mapaccessi moves 'it' to offset 'offi' in overflow bucket 'over' of bucket
// 'bucket' in hmap, which may or may not contain valid data. It returns true
// if the data is valid, and false otherwise.
func mapaccessi(t *maptype, h *hmap, it *hiter, bucket uintptr, over, offi uint8) bool {
	// grab snapshot of bucket state
	if t.bucket.kind&kindNoPointers != 0 {
		// Allocate the current slice and remember pointers to both current and old.
		// This preserves all relevant overflow buckets alive even if
		// the table grows and/or overflow buckets are added to the table
		// while we are iterating.
		h.createOverflow()
		it.overflow = *h.overflow
	}

	b := (*bmap)(add(h.buckets, bucket*uintptr(t.bucketsize)))

	checkBucket := false
	if h.oldbuckets != nil {
		// Iterator was started in the middle of a grow, and the grow isn't done yet.
		// If the bucket we're looking at hasn't been filled in yet (i.e. the old
		// bucket hasn't been evacuated) then we need to use that pointer instead.
		oldbucket := bucket & (uintptr(1)<<(h.B-1) - 1)
		oldB := (*bmap)(add(h.oldbuckets, oldbucket*uintptr(t.bucketsize)))
		if !evacuated(oldB) {
			b = oldB
			checkBucket = true
		}
	}

	// seek to overflow bucket
	for i := uint8(0); i < over; i++ {
		b = b.overflow(t)
		if b == nil {
			return false
		}
	}

	// check that bucket is not empty
	if b.tophash[offi] == empty || b.tophash[offi] == evacuatedEmpty {
		return false
	}

	// grab the key and value
	k := add(unsafe.Pointer(b), dataOffset+uintptr(offi)*uintptr(t.keysize))
	v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+uintptr(offi)*uintptr(t.valuesize))
	if t.indirectkey {
		k = *((*unsafe.Pointer)(k))
	}
	if t.indirectvalue {
		v = *((*unsafe.Pointer)(v))
	}

	// if this is an old bucket, we need to check whether this key is destined
	// for the new bucket. Otherwise, we will have a 2x bias towards oldbucket
	// values, since two different bucket selections can result in the same
	// oldbucket.
	if checkBucket {
		if t.reflexivekey || t.key.alg.equal(k, k) {
			// If the item in the oldbucket is not destined for
			// the current new bucket in the iteration, skip it.
			hash := t.key.alg.hash(k, uintptr(h.hash0))
			if hash&(uintptr(1)<<h.B-1) != bucket {
				return false
			}
		} else {
			// Hash isn't repeatable if k != k (NaNs).  We need a
			// repeatable and randomish choice of which direction
			// to send NaNs during evacuation. We'll use the low
			// bit of tophash to decide which way NaNs go.
			if bucket>>(h.B-1) != uintptr(b.tophash[offi]&1) {
				return false
			}
		}
	}

	it.key = k
	it.value = v
	return true
}
