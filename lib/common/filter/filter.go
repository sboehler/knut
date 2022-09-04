package filter

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

func Default[T any](_ T) bool {
	return true
}
