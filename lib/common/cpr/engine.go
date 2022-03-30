package cpr

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Source generates elements.
type Source[T any] interface {
	Source(context.Context, *errgroup.Group) <-chan T
}

// Processor processes elements.
type Processor[T any] interface {
	Process2(context.Context, *errgroup.Group, <-chan T) <-chan T
}

// Sink consumes elements.
type Sink[T any] interface {
	Sink(context.Context, *errgroup.Group, <-chan T)
}

// Engine processes a pipeline.
type Engine[T any] struct {
	Source     Source[T]
	Sink       Sink[T]
	Processors []Processor[T]
}

// Process processes the pipeline in the engine.
func (eng *Engine[T]) Process(ctx context.Context) error {
	grp, ctx := errgroup.WithContext(ctx)

	ch := eng.Source.Source(ctx, grp)

	for _, pr := range eng.Processors {
		ch = pr.Process2(ctx, grp, ch)
	}

	eng.Sink.Sink(ctx, grp, ch)
	return grp.Wait()
}

// Add adds a processor.
func (eng *Engine[T]) Add(p Processor[T]) {
	eng.Processors = append(eng.Processors, p)
}

// Collector collects channel result into an array.
type Collector[T any] struct {
	Result []T
}

// Sink implements Sink.
func (c *Collector[T]) Sink(ctx context.Context, g *errgroup.Group, ch <-chan T) {
	g.Go(func() error {
		for {
			d, ok, err := Pop(ctx, ch)
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

// Source implements Source.
func (p *Producer[T]) Source(ctx context.Context, g *errgroup.Group) <-chan T {
	ch := make(chan T)
	g.Go(func() error {
		defer close(ch)
		for _, i := range p.Items {
			if err := Push(ctx, ch, i); err != nil {
				return err
			}
		}
		return nil
	})
	return ch
}

// RunTestEngine runs the processor in a test engine.
func RunTestEngine[T any](ctx context.Context, ps Processor[T], ts ...T) ([]T, error) {
	sink := new(Collector[T])
	eng := &Engine[T]{
		Source:     &Producer[T]{ts},
		Processors: []Processor[T]{ps},
		Sink:       sink,
	}
	if err := eng.Process(ctx); err != nil {
		return nil, err
	}
	return sink.Result, nil
}
