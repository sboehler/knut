package order

import (
	"time"

	"golang.org/x/exp/constraints"
)

type Ordering int

const (
	Smaller Ordering = iota
	Equal
	Greater
)

type Compare[T any] func(t1, t2 T) Ordering

func CompareOrdered[T constraints.Ordered](t1, t2 T) Ordering {
	if t1 == t2 {
		return Equal
	}
	if t1 < t2 {
		return Smaller
	}
	return Greater
}

func CompareTime(t1, t2 time.Time) Ordering {
	if t1 == t2 {
		return Equal
	}
	if t1.Before(t2) {
		return Smaller
	}
	return Greater
}

func Desc[T any](cmp Compare[T]) Compare[T] {
	return func(t1, t2 T) Ordering {
		return cmp(t2, t1)
	}
}

func Asc[T any](cmp Compare[T]) Compare[T] {
	return cmp
}

func CompareCombined[T any](cmp ...Compare[T]) Compare[T] {
	return func(t1, t2 T) Ordering {
		for _, c := range cmp {
			if o := c(t1, t2); o != Equal {
				return o
			}
		}
		return Equal
	}
}
