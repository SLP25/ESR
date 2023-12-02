package utils

func CastChan[T any, U any](from <-chan T) chan U {
	to := make(chan U)
	go func() {
		var val any
		for val = range from {
			to <- val.(U)
		}
	}()
	return to
}

func MapChan[T any, U any](from <-chan T, f func(T) U) chan U {
	to := make(chan U)
	go func() {
		for val := range from {
			to <- f(val)
		}
	}()
	return to
}