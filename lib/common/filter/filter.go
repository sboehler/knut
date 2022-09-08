package filter

import (
	"regexp"
)

type Filter[T any] func(T) bool

func Combine[T any](fs ...Filter[T]) Filter[T] {
	return func(t T) bool {
		for _, f := range fs {
			if !f(t) {
				return false
			}
		}
		return true
	}
}

func AllowAll[T any](_ T) bool {
	return true
}

type Named interface {
	Name() string
}

func ByName[T Named](r *regexp.Regexp) Filter[T] {
	if r == nil {
		return AllowAll[T]
	}
	return func(n T) bool {
		return r.MatchString(n.Name())
	}
}
