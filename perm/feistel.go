// Package perm provides a generator object which can be used to iterate
// through a permutation of a given range in constant space.
//
// The current implementation uses a Feistel network, with blake2b as the
// round function. Note that since the entropy seeding the round function is
// typically far smaller than the full permutation space, the generator cannot
// enumerate the full space in such cases. For example, even a 128-bit key can
// only enumerate the permutation space of 34 elements, since 2^128 < 35!.
// This is a known limitation of Feistel networks.
//
// For small n, it may be more efficient to enumerate [0,n) and permute the
// values in-place using the well-known Fisher-Yates algorithm (this is what
// math/rand.Perm uses). For convenience, such a function is provided in this
// package as well.
package perm

import (
	"hash"
	"math/rand"

	"github.com/minio/blake2b-simd"
)

// FisherYates returns a pseudorandom permutation of the integers [0,n).
func FisherYates(n int) []int { return rand.Perm(n) }

type feistelGenerator struct {
	nextPow4    uint32
	halfNumBits uint32
	leftMask    uint32
	rightMask   uint32
	seed        uint32
	numElems    uint32
	i           uint32

	hash  hash.Hash
	arena [32]byte
}

// NewGenerator returns a new Feistel network-based permutation generator.
func NewGenerator(numElems, seed uint32) *feistelGenerator {
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

		hash: blake2b.New256(),
	}
}

func (f *feistelGenerator) Next() (uint32, bool) {
	for f.i < f.nextPow4 {
		n := f.encryptIndex(f.i)
		f.i++
		if n < f.numElems {
			return n, true
		}
	}
	return 0, false
}

func (f *feistelGenerator) encryptIndex(index uint32) uint32 {
	// split index into left and right bits
	left := (index & f.leftMask) >> f.halfNumBits
	right := (index & f.rightMask)

	// do 4 Feistel rounds
	for i := uint32(0); i < 4; i++ {
		ki := f.seed + i
		left, right = right, left^f.round(right, ki)
	}

	// join left and right bits to form permuted index
	return (left << f.halfNumBits) | right
}

func (f *feistelGenerator) round(right uint32, subkey uint32) uint32 {
	data := f.arena[:8]
	data[0] = byte(right >> 24)
	data[1] = byte(right >> 16)
	data[2] = byte(right >> 8)
	data[3] = byte(right)
	data[4] = byte(subkey >> 24)
	data[5] = byte(subkey >> 16)
	data[6] = byte(subkey >> 8)
	data[7] = byte(subkey)
	f.hash.Reset()
	f.hash.Write(data[:8])
	data = f.hash.Sum(f.arena[:0])

	r := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	return r & f.rightMask
}
