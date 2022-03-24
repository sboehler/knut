package ast

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"golang.org/x/sync/errgroup"
)

// Source generates elements.
type Source interface {
	Pop(context.Context) (any, bool, error)
}

// Processor processes elements.
type Processor interface {
	Process(ctx context.Context, elem any, ok bool, next func(any) bool) error
}

// Sink consumes elements.
type Sink interface {
	Push(ctx context.Context, elem any, ok bool) error
}

// Engine processes a pipeline.
type Engine struct {
	Source     Source
	Sink       Sink
	Processors []Processor
}

// Process processes the pipeline in the engine.
func (eng *Engine) Process(ctx context.Context) error {
	ch := make(chan any)
	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		defer close(ch)
		for {
			elem, ok, err := eng.Source.Pop(ctx)
			if err != nil || !ok {
				return err
			}
			if cpr.Push(ctx, ch, elem) != nil {
				return nil
			}
		}
	})

	for _, pr := range eng.Processors {
		pr := pr
		prevCh := ch
		nextCh := make(chan any)
		next := func(elem any) bool {
			return cpr.Push(ctx, nextCh, elem) == nil
		}
		grp.Go(func() error {
			defer close(nextCh)
			for {
				elem, ok, err := cpr.Pop(ctx, prevCh)
				if err != nil {
					return nil
				}
				if err := pr.Process(ctx, elem, ok, next); err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}
		})
		ch = nextCh
	}

	grp.Go(func() error {
		for {
			elem, ok, err := cpr.Pop(ctx, ch)
			if err != nil {
				return nil
			}
			if err := eng.Sink.Push(ctx, elem, ok); err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
	})
	return grp.Wait()
}
