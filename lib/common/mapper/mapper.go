package mapper

type Mapper[T any] func(T) T

func Identity[T any](t T) T {
	return t
}

func Nil[P interface{ *T }, T any](P) P {
	return nil
}

func Combine[T any](ms ...Mapper[T]) Mapper[T] {
	return func(t T) T {
		for _, m := range ms {
			t = m(t)
		}
		return t
	}
}
