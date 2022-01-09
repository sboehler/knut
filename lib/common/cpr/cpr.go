// Package cpr contains concurrency primitives.
package cpr

import (
	"context"
	"sync"
)

// Get gets and returns a new T from the supplied channel. It returns
// a T, a boolean which indicates whether the channel is still open, or\
// an error from the supplied errCh.
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

// Pop returns a new T from the ch. It returns a boolean which indicates
// whether the channel is still open. The error indicates whether the context
// has been canceled.
func Pop[T any](ctx context.Context, ch <-chan T) (T, bool, error) {
	var res T
	select {
	case d, ok := <-ch:
		return d, ok, ctx.Err()
	case <-ctx.Done():
		return res, false, ctx.Err()
	}
}

// Push tries to push a T to the ch. The error indicates whether the context
// has been canceled.
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

// Demultiplex demultiplexes the given channels.
func Demultiplex[T any](inChs ...<-chan T) chan T {
	var (
		wg    sync.WaitGroup
		resCh = make(chan T)
	)
	wg.Add(len(inChs))
	for _, inCh := range inChs {
		go func(ch <-chan T) {
			defer wg.Done()
			for t := range ch {
				resCh <- t
			}
		}(inCh)
	}
	go func() {
		wg.Wait()
		close(resCh)
	}()
	return resCh
}
