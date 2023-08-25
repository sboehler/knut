package predicate

import (
	"github.com/sboehler/knut/lib/common/regex"
)

type Predicate[T any] func(T) bool

func And[T any](predicates ...Predicate[T]) Predicate[T] {
	return func(t T) bool {
		for _, pred := range predicates {
			if !pred(t) {
				return false
			}
		}
		return true
	}
}

func True[T any](_ T) bool {
	return true
}

type Named interface {
	Name() string
}

func ByName[T Named](rxs regex.Regexes) Predicate[T] {
	if len(rxs) == 0 {
		return True[T]
	}
	return func(t T) bool {
		return rxs.MatchString(t.Name())
	}
}

func Or[T any](fs ...Predicate[T]) Predicate[T] {
	return func(t T) bool {
		for _, f := range fs {
			if f(t) {
				return true
			}
		}
		return false
	}
}

func Not[T any](f Predicate[T]) Predicate[T] {
	return func(t T) bool {
		return !f(t)
	}
}
