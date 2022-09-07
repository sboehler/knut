package dict

import (
	"sort"

	"github.com/sboehler/knut/lib/common/compare"
)

func Keys[K comparable, V any](m map[K]V) []K {
	res := make([]K, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
}

func SortedKeys[K comparable, V any](m map[K]V, c compare.Compare[K]) []K {
	res := Keys(m)
	sort.Slice(res, func(i, j int) bool {
		return c(res[i], res[j]) == compare.Smaller
	})
	return res
}

func Values[K comparable, V any](m map[K]V) []V {
	res := make([]V, 0, len(m))
	for _, v := range m {
		res = append(res, v)
	}
	return res
}

func SortedValues[K comparable, V any](m map[K]V, c compare.Compare[V]) []V {
	res := Values(m)
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
