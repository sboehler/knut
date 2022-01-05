// Package cpr contains concurrency primitives.
package cpr

func Get[T any](ch <-chan T, errCh <-chan error) (T, bool, error) {
	for {
		select {
		case d, ok := <-ch:
			return d, ok, nil
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				break
			}
			var def T
			return def, ok, err
		}
	}
}
