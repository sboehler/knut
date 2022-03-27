package ast

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"golang.org/x/sync/errgroup"
)

// Source generates elements.
type Source interface {
	Pop(context.Context) (Dated, bool, error)
}

// Processor processes elements.
type Processor interface {
	Process(ctx context.Context, elem Dated, ok bool, next func(Dated) bool) error
}

// Sink consumes elements.
type Sink interface {
	Push(ctx context.Context, elem Dated, ok bool) error
}

// Engine processes a pipeline.
type Engine struct {
	Source     Source
	Sink       Sink
	Processors []Processor
}

// Process processes the pipeline in the engine.
func (eng *Engine) Process(ctx context.Context) error {
	ch := make(chan Dated)
	grp, ctx := errgroup.WithContext(ctx)
	{
		ch := ch
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
	}

	for _, pr := range eng.Processors {
		pr := pr
		prevCh := ch
		nextCh := make(chan Dated)
		ch = nextCh
		next := func(elem Dated) bool {
			return cpr.Push(ctx, nextCh, elem) == nil
		}
		grp.Go(func() error {
			defer func() {
				close(nextCh)
			}()
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
	}

	{
		ch := ch
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
	}
	return grp.Wait()
}

// Add adds a processor.
func (eng *Engine) Add(p Processor) {
	eng.Processors = append(eng.Processors, p)
}
