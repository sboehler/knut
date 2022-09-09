package mapper

type Mapper[T any] func(T) T

func Identity[T any](t T) T {
	return t
}

func Nil[P interface{ *T }, T any](P) P {
	return nil
}
