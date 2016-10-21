package randmap

import "testing"

func TestKey(t *testing.T) {
	m := map[int]int{
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		0: 0,
	}
	counts := make([]int, len(m))
	counts2 := make([]int, len(m))
	for i := 0; i < 100; i++ {
		counts[Key(m).(int)]++
	}
	for i := 0; i < 100; i++ {
		for n := range m {
			counts2[n]++
			break
		}
	}
	for n, c := range counts {
		if (100/len(m))/2 > c || c > (100/len(m))*2 {
			t.Errorf("suspicious count: expected %v-%v, got %v (%v)", (100/len(m))/2, (100/len(m))*2, c, n)
		}
		t.Log(n, c)
	}

	t.Log("default")

	for n, c := range counts2 {
		if (100/len(m))/2 > c || c > (100/len(m))*2 {
			t.Errorf("suspicious default count: expected %v-%v, got %v (%v)", (100/len(m))/2, (100/len(m))*2, c, n)
		}
		t.Log(n, c)
	}
}

func TestVal(t *testing.T) {
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < 100; i++ {
		counts[Val(m).(int)]++
	}
	for _, c := range counts {
		if (100/len(m))/2 > c || c > (100/len(m))*2 {
			t.Fatalf("suspicious count: expected %v-%v, got %v", (100/len(m))/2, (100/len(m))*2, c)
		}
	}
}

func TestFastKey(t *testing.T) {
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < 100; i++ {
		counts[FastKey(m).(int)]++
	}
	for _, c := range counts {
		if (100/len(m))/2 > c || c > (100/len(m))*2 {
			t.Fatalf("suspicious count: expected %v-%v, got %v", (100/len(m))/2, (100/len(m))*2, c)
		}
	}
}

func TestFastVal(t *testing.T) {
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	counts := make([]int, len(m))
	for i := 0; i < 100; i++ {
		counts[FastVal(m).(int)]++
	}
	for _, c := range counts {
		if (100/len(m))/2 > c || c > (100/len(m))*2 {
			t.Fatalf("suspicious count: expected %v-%v, got %v", (100/len(m))/2, (100/len(m))*2, c)
		}
	}
}

func TestPtr(t *testing.T) {
	m := map[string]string{
		"0": "0",
		"1": "1",
		"2": "2",
		"3": "3",
		"4": "4",
	}
	for i := 0; i < len(m); i++ {
		println(Key(m).(string), Val(m).(string))
	}
}
