// Package cpr contains concurrency primitives.
package cpr

import (
	"context"
	"sync"

	"github.com/sourcegraph/conc/pool"
)

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

func Parallel(fs ...func()) func() {
	var wg sync.WaitGroup
	wg.Add(len(fs))
	for _, f := range fs {
		f := f
		go func() {
			f()
			wg.Done()
		}()
	}
	return wg.Wait
}

func ForAll[T any](ts []T, f func(T)) func() {
	var wg sync.WaitGroup
	wg.Add(len(ts))
	for _, t := range ts {
		go func(t T) {
			f(t)
			wg.Done()
		}(t)
	}
	return wg.Wait
}

func ForEach[T any](ctx context.Context, ch <-chan T, f func(T) error) error {
	for {
		t, ok, err := Pop(ctx, ch)
		if err != nil || !ok {
			return err
		}
		if err := f(t); err != nil {
			return err
		}
	}
}

func Produce[T any](f func(context.Context, chan<- T) error) (<-chan T, func(context.Context) error) {
	ch := make(chan T)
	return ch, func(ctx context.Context) error {
		defer close(ch)
		return f(ctx, ch)
	}
}

func Seq[T any](ctx context.Context, ts []T, fs ...func(T) error) ([]T, error) {
	var workers []func(context.Context) error
	prevCh, w := Produce(func(ctx context.Context, ch chan<- T) error {
		for _, t := range ts {
			if err := Push(ctx, ch, t); err != nil {
				return err
			}
		}
		return nil
	})
	workers = append(workers, w)
	for _, f := range fs {
		f, inCh := f, prevCh
		ch, w := Produce(func(ctx context.Context, ch chan<- T) error {
			return ForEach(ctx, inCh, func(t T) error {
				if err := f(t); err != nil {
					return err
				}
				return Push(ctx, ch, t)
			})
		})
		workers = append(workers, w)
		prevCh = ch
	}

	ch, w := FanIn(func(ctx context.Context, ch chan<- []T) error {
		var res []T
		err := ForEach(ctx, prevCh, func(t T) error {
			res = append(res, t)
			return nil
		})
		if err != nil {
			return err
		}
		return Push(ctx, ch, res)
	})

	workers = append(workers, w)
	p := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()
	for _, w := range workers {
		p.Go(w)
	}
	if err := p.Wait(); err != nil {
		return nil, err
	}
	return <-ch, nil
}

func FanIn[T any](f func(context.Context, chan<- T) error) (<-chan T, func(context.Context) error) {
	ch := make(chan T, 1)
	return ch, func(ctx context.Context) error {
		defer close(ch)
		return f(ctx, ch)
	}
}
