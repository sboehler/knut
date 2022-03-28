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
	Process(ctx context.Context, elem Dated, next func(Dated) bool) error
}

// Sink consumes elements.
type Sink interface {
	Push(ctx context.Context, elem Dated) error
}

// FinalizeSink has a method which is called after the last
// element has been received.
type FinalizeSink interface {
	Finalize(ctx context.Context) error
}

// Finalize has a method which is called after the last element
// has been received.
type Finalize interface {
	Finalize(ctx context.Context, next func(Dated) bool) error
}

// Engine processes a pipeline.
type Engine struct {
	Source     Source
	Sink       Sink
	Processors []Processor
}

const channelBuffer = 1000

// Process processes the pipeline in the engine.
func (eng *Engine) Process(ctx context.Context) error {
	ch := make(chan Dated, channelBuffer)
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
		pr, prevCh, nextCh := pr, ch, make(chan Dated, channelBuffer)
		ch = nextCh
		next := func(elem Dated) bool {
			return cpr.Push(ctx, nextCh, elem) == nil
		}
		grp.Go(func() error {
			defer close(nextCh)
			for {
				elem, ok, err := cpr.Pop(ctx, prevCh)
				if err != nil {
					return err
				}
				if !ok {
					if f, ok := pr.(Finalize); ok {
						return f.Finalize(ctx, next)
					}
					return nil
				}
				if err := pr.Process(ctx, elem, next); err != nil {
					return err
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
				if !ok {
					if f, ok := eng.Sink.(FinalizeSink); ok {
						return f.Finalize(ctx)
					}
					return nil
				}
				if err := eng.Sink.Push(ctx, elem); err != nil {
					return err
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
