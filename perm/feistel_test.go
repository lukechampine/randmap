package perm

import (
	"math/rand"
	"testing"
)

func TestGenerator(t *testing.T) {
	const numElems = 50
	const iters = 10000
	counts := make([][]int, numElems)
	for i := range counts {
		counts[i] = make([]int, numElems)
	}
	for i := 0; i < iters; i++ {
		g := NewGenerator(numElems, rand.Uint32())
		for j := 0; ; j++ {
			u, ok := g.Next()
			if !ok {
				break
			}
			// u appeared at index j
			counts[u][j]++
		}
	}

	// each key should have appeared at each index about iters/numElems times
	for k, cs := range counts {
		for i, c := range cs {
			if (iters/numElems)/2 > c || c > (iters/numElems)*2 {
				t.Errorf("suspicious count for key %v index %v: expected %v-%v, got %v", k, i, (iters/numElems)/2, (iters/numElems)*2, c)
			}
		}
	}
}
