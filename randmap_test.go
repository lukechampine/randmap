package randmap

import "testing"

func TestBuiltinMap(t *testing.T) {
	const iters = 100000
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < iters; i++ {
		for n := range m {
			counts[n]++
			break
		}
	}
	// 0 should be "randomly selected" 45-55% of the time
	if (iters/2-iters/20) > counts[0] || counts[0] > (iters/2+iters/20) {
		t.Errorf("expected builtin map to be less random: expected ~%v for elem 0, got %v", iters/2, counts[0])
	}
}

func TestKey(t *testing.T) {
	const iters = 100000
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < iters; i++ {
		counts[Key(m).(int)]++
	}

	for n, c := range counts {
		if (iters/len(m))/2 > c || c > (iters/len(m))*2 {
			t.Errorf("suspicious count: expected %v-%v, got %v (%v)", (iters/len(m))/2, (iters/len(m))*2, c, n)
		}
	}
}

func TestVal(t *testing.T) {
	const iters = 100000
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < iters; i++ {
		counts[Val(m).(int)]++
	}

	for n, c := range counts {
		if (iters/len(m))/2 > c || c > (iters/len(m))*2 {
			t.Errorf("suspicious count: expected %v-%v, got %v (%v)", (iters/len(m))/2, (iters/len(m))*2, c, n)
		}
	}
}

func TestFastKey(t *testing.T) {
	const iters = 100000
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < iters; i++ {
		counts[FastKey(m).(int)]++
	}

	for n, c := range counts {
		if (iters/len(m))/2 > c || c > (iters/len(m))*2 {
			t.Errorf("suspicious count: expected %v-%v, got %v (%v)", (iters/len(m))/2, (iters/len(m))*2, c, n)
		}
	}
}

func TestFastVal(t *testing.T) {
	const iters = 100000
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < iters; i++ {
		counts[FastVal(m).(int)]++
	}

	for n, c := range counts {
		if (iters/len(m))/2 > c || c > (iters/len(m))*2 {
			t.Errorf("suspicious count: expected %v-%v, got %v (%v)", (iters/len(m))/2, (iters/len(m))*2, c, n)
		}
	}
}

func TestEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when accessing empty map")
		}
	}()
	_ = Key(make(map[int]int))
}

func BenchmarkKey(b *testing.B) {
	m := make(map[int]int, 10000)
	for i := 0; i < 10000; i++ {
		m[i] = i
	}

	b.Run("key", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Key(m).(int)
		}
	})

	b.Run("fastkey", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = FastKey(m).(int)
		}
	})

	b.Run("stdlib", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for n := range m {
				_ = n
				break
			}
		}
	})
}
