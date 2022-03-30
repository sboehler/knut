package ast

import (
	"context"

	"github.com/sboehler/knut/lib/common/cpr"
	"golang.org/x/sync/errgroup"
)

// Source2 generates elements.
type Source2[T any] interface {
	Source2(context.Context, *errgroup.Group) <-chan T
}

// Processor2 processes elements.
type Processor2[T any] interface {
	Process2(context.Context, *errgroup.Group, <-chan T) <-chan T
}

// Sink2 consumes elements.
type Sink2[T any] interface {
	Sink2(context.Context, *errgroup.Group, <-chan T)
}

// Engine2 processes a pipeline.
type Engine2[T any] struct {
	Source     Source2[T]
	Sink       Sink2[T]
	Processors []Processor2[T]
}

// Process processes the pipeline in the engine.
func (eng *Engine2[T]) Process(ctx context.Context) error {
	grp, ctx := errgroup.WithContext(ctx)

	ch := eng.Source.Source2(ctx, grp)

	for _, pr := range eng.Processors {
		ch = pr.Process2(ctx, grp, ch)
	}

	eng.Sink.Sink2(ctx, grp, ch)
	return grp.Wait()
}

// Add adds a processor.
func (eng *Engine2[T]) Add(p Processor2[T]) {
	eng.Processors = append(eng.Processors, p)
}

// Collector collects channel result into an array.
type Collector[T any] struct {
	Result []T
}

// Sink2 implements Sink.
func (c *Collector[T]) Sink2(ctx context.Context, g *errgroup.Group, ch <-chan T) {
	g.Go(func() error {
		for {
			d, ok, err := cpr.Pop(ctx, ch)
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			c.Result = append(c.Result, d)
		}
		return nil
	})
}

// Producer produces values.
type Producer[T any] struct {
	Items []T
}

// Source2 implements Source.
func (p *Producer[T]) Source2(ctx context.Context, g *errgroup.Group) <-chan T {
	ch := make(chan T)
	g.Go(func() error {
		defer close(ch)
		for _, i := range p.Items {
			if err := cpr.Push(ctx, ch, i); err != nil {
				return err
			}
		}
		return nil
	})
	return ch
}

// RunTestEngine runs the processor in a test engine.
func RunTestEngine[T any](ctx context.Context, ps Processor2[T], ts ...T) ([]T, error) {
	sink := new(Collector[T])
	eng := &Engine2[T]{
		Source:     &Producer[T]{ts},
		Processors: []Processor2[T]{ps},
		Sink:       sink,
	}
	if err := eng.Process(ctx); err != nil {
		return nil, err
	}
	return sink.Result, nil
}
