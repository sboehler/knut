// Package cpr contains concurrency primitives.
package cpr

import "context"

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

func Pop[T any](ctx context.Context, ch <-chan T) (T, bool, error) {
	var res T
	select {
	case d, ok := <-ch:
		return d, ok, ctx.Err()
	case <-ctx.Done():
		return res, false, ctx.Err()
	}
}

func Push[T any](ctx context.Context, ch chan<- T, ts ...T) error {
	for _, t := range ts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- t:
		}
	}
	return nil
}
