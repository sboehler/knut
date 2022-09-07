package maputils

import (
	"sort"

	"github.com/sboehler/knut/lib/common/compare"
)

func SortedKeys[K comparable, V any](m map[K]V, c compare.Compare[K]) []K {
	res := make([]K, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool {
		return c(res[i], res[j]) == compare.Smaller
	})
	return res
}

func SortedValues[K comparable, V any](m map[K]V, c compare.Compare[V]) []V {
	res := make([]V, 0, len(m))
	for _, k := range m {
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
