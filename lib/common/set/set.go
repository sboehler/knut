package set

import "github.com/sboehler/knut/lib/common/compare"

type Set[T comparable] map[T]struct{}

func New[T comparable]() Set[T] {
	return make(Set[T])
}

func (set Set[T]) Add(t T) {
	set[t] = struct{}{}
}

func (set Set[T]) Has(t T) bool {
	_, ok := set[t]
	return ok
}

func (set Set[T]) Remove(t T) {
	delete(set, t)
}

func (set Set[T]) Slice() []T {
	res := make([]T, 0, len(set))
	for elem := range set {
		res = append(res, elem)
	}
	return res
}

func (set Set[T]) Sorted(cmp compare.Compare[T]) []T {
	res := set.Slice()
	compare.Sort(res, cmp)
	return res
}

func Of[T comparable](ts ...T) Set[T] {
	res := New[T]()
	for _, t := range ts {
		res.Add(t)
	}
	return res
}
