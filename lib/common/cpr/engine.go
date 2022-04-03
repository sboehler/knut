package cpr

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Source generates elements.
type Source[T any] interface {
	Source(context.Context, chan<- T) error
}

// Processor processes elements.
type Processor[T any] interface {
	Process(context.Context, <-chan T, chan<- T) error
}

// Sink consumes elements.
type Sink[T any] interface {
	Sink(context.Context, <-chan T) error
}

// Engine processes a pipeline.
type Engine[T any] struct {
	Source     Source[T]
	Sink       Sink[T]
	Processors []Processor[T]
}

const bufSize = 100

// Process processes the pipeline in the engine.
func (eng *Engine[T]) Process(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, bufSize)
	{
		outCh := ch
		g.Go(func() error {
			defer close(outCh)
			return eng.Source.Source(ctx, outCh)
		})
	}
	for _, pr := range eng.Processors {
		pr, inCh, outCh := pr, ch, make(chan T, bufSize)
		g.Go(func() error {
			defer close(outCh)
			return pr.Process(ctx, inCh, outCh)
		})
		ch = outCh
	}
	{
		inCh := ch
		g.Go(func() error {
			return eng.Sink.Sink(ctx, inCh)
		})
	}
	return g.Wait()
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
func (c *Collector[T]) Sink(ctx context.Context, inCh <-chan T) error {
	for {
		d, ok, err := Pop(ctx, inCh)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		c.Result = append(c.Result, d)
	}
	return nil
}

// Producer produces values.
type Producer[T any] struct {
	Items []T
}

// Source implements Source.
func (p *Producer[T]) Source(ctx context.Context, outCh chan<- T) error {
	for _, i := range p.Items {
		if err := Push(ctx, outCh, i); err != nil {
			return err
		}
	}
	return nil
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
