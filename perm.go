package randmap

import (
	"hash"

	"github.com/dchest/blake2b"
)

type feistelGenerator struct {
	nextPow4    uint32
	halfNumBits uint32
	leftMask    uint32
	rightMask   uint32
	seed        uint32
	numElems    uint32

	hash  hash.Hash
	arena []byte
}

func newGenerator(numElems, seed uint32) *feistelGenerator {
	nextPow4 := uint32(4)
	log4 := uint32(1)
	for nextPow4 < numElems {
		nextPow4 *= 4
		log4++
	}

	return &feistelGenerator{
		nextPow4:    nextPow4,
		halfNumBits: log4,
		leftMask:    ((uint32(1) << log4) - 1) << log4, // e.g. 0xFFFF0000
		rightMask:   (uint32(1) << log4) - 1,           // e.g. 0x0000FFFF
		seed:        seed,
		numElems:    numElems,

		hash:  blake2b.New256(),
		arena: make([]byte, 32),
	}
}

func (f *feistelGenerator) Iter(fn func(u uint32)) {
	for i := uint32(0); i < f.nextPow4; i++ {
		n := f.encryptIndex(i)
		if n < f.numElems {
			fn(n)
		}
	}
}

func (f *feistelGenerator) encryptIndex(index uint32) uint32 {
	// split index into left and right bits
	left := (index & f.leftMask) >> f.halfNumBits
	right := (index & f.rightMask)

	// do 4 Feistel rounds
	for i := uint32(0); i < 4; i++ {
		r := round(f.hash, f.arena, f.seed+i, right) & f.rightMask
		left, right = right, left^r
	}

	// join left and right bits to form permuted index
	return (left << f.halfNumBits) | right
}

func round(h hash.Hash, arena []byte, seed uint32, data uint32) uint32 {
	arena[0] = byte(seed >> 24)
	arena[1] = byte(seed >> 16)
	arena[2] = byte(seed >> 8)
	arena[3] = byte(seed)
	arena[4] = byte(data >> 24)
	arena[5] = byte(data >> 16)
	arena[6] = byte(data >> 8)
	arena[7] = byte(data)
	h.Reset()
	h.Write(arena[:8])
	arena = h.Sum(arena[:0])

	r := uint32(arena[0])<<24 | uint32(arena[1])<<16 | uint32(arena[2])<<8 | uint32(arena[3])
	return r
}
