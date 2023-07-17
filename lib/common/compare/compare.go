package compare

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/constraints"
)

type Order int

const (
	Smaller Order = -1
	Equal   Order = 0
	Greater Order = 1
)

type Compare[T any] func(t1, t2 T) Order

func Ordered[T constraints.Ordered](t1, t2 T) Order {
	if t1 < t2 {
		return Smaller
	}
	if t1 == t2 {
		return Equal
	}
	return Greater
}

func Time(t1, t2 time.Time) Order {
	if t1 == t2 {
		return Equal
	}
	if t1.Before(t2) {
		return Smaller
	}
	return Greater
}

func Decimal(t1, t2 decimal.Decimal) Order {
	if t1.Equal(t2) {
		return Equal
	}
	if t1.LessThan(t2) {
		return Smaller
	}
	return Greater
}

func Desc[T any](cmp Compare[T]) Compare[T] {
	return func(t1, t2 T) Order {
		return cmp(t2, t1)
	}
}

func Asc[T any](cmp Compare[T]) Compare[T] {
	return cmp
}

func Combine[T any](cmp ...Compare[T]) Compare[T] {
	return func(t1, t2 T) Order {
		for _, c := range cmp {
			if o := c(t1, t2); o != Equal {
				return o
			}
		}
		return Equal
	}
}

func Sort[T any](ts []T, cmp func(T, T) Order) {
	sort.Slice(ts, func(i, j int) bool {
		return cmp(ts[i], ts[j]) == Smaller
	})
}
