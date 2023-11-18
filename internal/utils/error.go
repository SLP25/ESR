package utils

import "log/slog"

func Warn(err error) {
	if err != nil {
		slog.Warn(err.Error())
	}
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