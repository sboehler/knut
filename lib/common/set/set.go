package set

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