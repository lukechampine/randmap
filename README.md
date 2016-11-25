randmap
-------

[![GoDoc](https://godoc.org/github.com/lukechampine/randmap?status.svg)](https://godoc.org/github.com/lukechampine/randmap)
[![Go Report Card](http://goreportcard.com/badge/github.com/lukechampine/randmap)](https://goreportcard.com/report/github.com/lukechampine/randmap)

```
go get github.com/lukechampine/randmap
```

randmap provides methods for (efficiently) accessing random elements of maps, and iterating
through maps in random order. [Here is a blog post](http://lukechampine.com/hackmap.html) that dives deeper into how I
accomplished this, if you're interested in that sort of thing.

**WARNING:** randmap uses the `unsafe` package to access the internal Go map
type. In general, you should think twice before running any code that imports
`unsafe`. Please read the full README (and the code!) if you are considering
using randmap in any serious capacity. If you want the `randmap` functionality
without the risks, you can import the `randmap/safe` package instead, which
is far less efficient but does not use `unsafe`.

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

How frequently do you think 0 will be selected, and how frequently will 1 be
selected? The answer is that 0 will be selected **7x** more frequently than 1!
So we can conclude that map iteration does not produce uniformly random
elements. (Actually, it's even worse than that; for certain map states, there
are elements that `range` will _never_ start on!)

This leaves us with only two options: we can flatten the map and select a
random index, or we can iterate a random number of times. But these incur
costs of O(_n_) memory and O(_n_) time, respectively. That might be ok for
small maps, but maps aren't always small.

Neither of these options were acceptable to me, so I now present a third
option: random map access in constant space and time. Specifically, this
approach uses O(1) space and O(_L_ * (1 + _k_/_n_)) time, where _n_ is the
number of elements in the map, _k_ is the "capacity" of the map (how many
elements it can hold before being resized), and _L_ is the length of the
longest "bucket chain." Since Go maps double in size when they grow, the total
time will generally not exceed 2x the normal map lookup time.

The algorithm is as follows: we begin the same way as the builtin algorithm,
by selecting a random index. If the index contains an element, we return it.
If it is empty, we try another random index. (The builtin algorithm seeks
forward until it hits a non-empty index.) That's it! In theory, this approach
actually has an unbounded run time (because you may select the same index
twice), but in practice this is rare. I may improve the implementation later
to iterate through a random permutation of indices, which would guarantee
eventual termination.

## Random Iteration ##

randmap provides `Iter` and `FastIter` functions for iterating through maps in
random or pseudeorandom order. The standard way to do this is to flatten the
map and use `math/rand.Perm` to iterate through the slice. Although this is
sufficiently random, it requires O(_n_) space and time, which is infeasible
for large maps. randmap instead uses a Feistel network to simultaneously
generate and iterate through permutations in constant space. This approach is
further detailed in the docstring of the `perm` subpackage. The main tradeoff
is that the generator approach will be much slower when _n_ is small.

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
var k, v int
i := randmap.Iter(m, &k, &v)
for i.Next() {
	// use k and v
}

// iterate in pseudorandom order
i := randmap.FastIter(m, &k, &v)
for i.Next() {
	// use k and v
}
```

In case it wasn't obvious, `Key`/`Val`/`Iter` use `crypto/rand`, while their
`Fast` equivalents use `math/rand`. You should use the former in any code that
requires cryptographically strong randomness.

## Caveats ##

This package obviously depends heavily on the internal representation of the
`map` type. If it changes, this package may break. By the way, the `map` type
is changing in Go 1.8. As stated above, use the `randmap/safe` package if you
want the functionality of `randmap` without the risks.

The runtime code governing maps is a bit esoteric, and uses constructs that
aren't available outside of the runtime. Concurrent map operations are
especially tricky. For now, no guarantees are made about concurrent use of the
functions in this package. Guarding map accesses with a mutex should be
sufficient to prevent any problems.

The provided Iterators are not guaranteed to uniformly cover the full
permutation space of a given map. This is because the number of permutations
may be much larger than the entropy of the iterator's seed. Nevertheless, the
seed is sufficient to prevent an attacker from guessing which permutation was
selected.
