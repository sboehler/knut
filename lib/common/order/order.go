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

func Ordered[T constraints.Ordered](t1, t2 T) Ordering {
	if t1 == t2 {
		return Equal
	}
	if t1 < t2 {
		return Smaller
	}
	return Greater
}

func Time(t1, t2 time.Time) Ordering {
	if t1 == t2 {
		return Equal
	}
	if t1.Before(t2) {
		return Smaller
	}
	return Greater
}
