package utils

type ServiceType byte

const (
	Bootstrapper ServiceType = iota
	Client
	Node
	Server
)


func CastChan[T any, U any](from chan T) chan U {
	to := make(chan U)
	go func() {
		var val any
		for {
			val = <-from
			to <- val.(U)
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