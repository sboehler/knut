package maputils

import (
	"sort"

	"github.com/sboehler/knut/lib/common/compare"
)

func SortedKeys[T comparable, V any](m map[T]V, c compare.Compare[T]) []T {
	res := make([]T, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool {
		return c(res[i], res[j]) == compare.Smaller
	})
	return res
}

func GetDefault[K comparable, V any](m map[K]V, k K, c func() V) V {
	v, ok := m[k]
	if !ok {
		v = c()
		m[k] = v
	}
	return v
}
