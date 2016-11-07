randmap
-------

[![GoDoc](https://godoc.org/github.com/lukechampine/randmap?status.svg)](https://godoc.org/github.com/lukechampine/randmap)
[![Go Report Card](http://goreportcard.com/badge/github.com/lukechampine/randmap)](https://goreportcard.com/report/github.com/lukechampine/randmap)

```
go get github.com/lukechampine/randmap
```

randmap provides methods for accessing random elements of maps, and iterating
through maps in random order.

First, it is important to clear up a misconception about Go's map type: that
`range` iterates through maps in random order. Well, what does the Language
Specification say?

>The iteration order over maps is not specified and is not guaranteed to be
>the same from one iteration to the next. [[1]](https://golang.org/ref/spec#For_statements)

This says nothing at all about whether the iteration is random! In fact, you
can confirm for yourself that iteration order is not very random at all. The
(current) map iterator implementation simply chooses a random index in the map
data and iterates forward from there, skipping empty "buckets" and wrapping
around at the end. This can lead to surprising behavior:

```go
m := map[int]int{
	0: 0,
	1: 1,
}
for i := range m {
	fmt.Printf("selected %v!\n", i)
	break
}
```

How frequently do you think `0` will be selected, and how frequently will `1`
be selected? The answer is that `0` will be selected **7x** more frequently
than `1`! So we cannot rely on map iteration as a means of choosing a random
element. (Actually, it's even worse than that; for certain map states, there
are elements that `range` will _never_ start on!)

This leaves us with only two options: we can flatten the map and select a
random index, or we can iterate a random number of times. But these incur
costs of O(_n_) memory and O(_n_) time, respectively. That might be ok for
small maps, but maps aren't always small.

Neither of these options were acceptable to me, so I now present a third
option: random map access in constant space and time. Specifically, this
approach uses O(1) space and O(_k_/_n_ * _t_) time, where _n_ is the number of
elements in the map, _k_ is the "capacity" of the map (how many elements it
can hold before being resized), and _t_ is the overhead incurred by a standard
map iteration. Since Go maps double in size when they grow, the first factor
should generally be <=2, meaning the total time will not exceed 2x the normal
map lookup time.

The algorithm is as follows: we begin the same way as the builtin algorithm,
by selecting a random index. If the index contains an element, we return it.
If it is empty, we try another random index. (The builtin algorithm seeks
forward until it hits a non-empty index.) That's it! In theory, this approach
actually has an unbounded run time (because you may select the same index
twice), but in practice this is rare. I may improve the implementation later
to iterate through a random permutation of indices, which would guarantee
eventual termination.

## Examples ##

```go
m := map[int]int{
	0: 0,
	1: 1,
}

// select a random key
k := randmap.Key(m).(int)

// select a random value
v := randmap.Val(m).(int)

// select a pseudorandom key
k := randmap.FastKey(m).(int)

// select a pseudorandom value
v := randmap.FastVal(m).(int)

// iterate in random order
randmap.Iter(m, func(k, v int) {
	// use k and v
})

// iterate in pseudorandom order
randmap.FastIter(m, func(k, v int) {
	// use k and v
})
```

In case it wasn't obvious, `Key`/`Val`/`Iter` use `crypto/rand`, while their
`Fast` equivalents use `math/rand`. You should use the former in any code that
requires cryptographically strong randomness.

## Caveats ##

This package obviously depends heavily on the internal representation of the
`map` type. If it changes, this package may break.

The runtime code governing maps is a bit esoteric, and uses constructs that
aren't available outside of the runtime. Concurrent map operations are
especially tricky. For now, no guarantees are made about concurrent use of the
functions in this package. Guarding map accesses with a mutex should be
sufficient to prevent any problems.
