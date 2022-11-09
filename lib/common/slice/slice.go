package slice

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"golang.org/x/sync/errgroup"
)

func Parallel[T any](ctx context.Context, ts []T, fs ...func(T) error) error {
	if len(fs) == 0 {
		return nil
	}
	wg, ctx := errgroup.WithContext(ctx)
	firstCh := make(chan T)
	ch := firstCh
	wg.Go(func() error {
		defer close(firstCh)
		for _, t := range ts {
			if err := cpr.Push(ctx, firstCh, t); err != nil {
				return err
			}
		}
		return nil
	})
	for _, f := range fs[:len(fs)-1] {
		inCh, f := ch, f
		outCh := make(chan T)
		ch = outCh
		wg.Go(func() error {
			defer close(outCh)
			return cpr.Consume(ctx, inCh, func(t T) error {
				if err := f(t); err != nil {
					return err
				}
				return cpr.Push(ctx, outCh, t)
			})
		})
	}
	wg.Go(func() error {
		return cpr.Consume(ctx, ch, fs[len(fs)-1])
	})
	return wg.Wait()
}
