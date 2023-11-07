package utils

type ServiceType byte

const (
	Bootstrapper ServiceType = iota
	Client
	Node
	Server
)


func CastChan[T any, U any](from <-chan T) chan U {
	to := make(chan U)
	go func() {
		var val any
		for val = range from {
			val = <-from
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


type risky func() error

func ChainError(funcs ...risky) error {
	for _, f := range funcs {
        err := f()
		if err != nil {
			return err
		}
    }
	return nil
}

func ContainsKey[K comparable, V any](dict map[K]V, field K) bool {
	_, ok := dict[field]
	return ok
}

func GetAnyKey[K comparable, V any](dict map[K]V, defaultKey K) K {
	for k, _ := range dict {
		return k
	}

	return defaultKey
}

func GetAnyValue[K comparable, V any](dict map[K]V, defaultVal V) V {
	for _, v := range dict {
		return v
	}

	return defaultVal
}