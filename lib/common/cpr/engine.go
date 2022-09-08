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

const bufSize = 100

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
func (c *Producer[T]) Source(ctx context.Context, outCh chan<- T) error {
	for _, i := range c.Items {
		if err := Push(ctx, outCh, i); err != nil {
			return err
		}
	}
	return nil
}

// RunTestEngine runs the processor in a test engine.
func RunTestEngine[T any](ctx context.Context, ps Processor[T], ts ...T) ([]T, error) {
	sink := new(Collector[T])
	ppl := Connect[T](Compose[T](&Producer[T]{ts}, ps), sink)
	if err := ppl.Process(ctx); err != nil {
		return nil, err
	}
	return sink.Result, nil
}

// Pipeline represents a computation.
type Pipeline interface {
	Process(context.Context) error
}

type source[T any] struct {
	source Source[T]
	proc   Processor[T]
}

func (c source[T]) Source(ctx context.Context, oCh chan<- T) error {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, bufSize)
	g.Go(func() error {
		defer close(ch)
		return c.source.Source(ctx, ch)
	})
	g.Go(func() error {
		return c.proc.Process(ctx, ch, oCh)
	})
	return g.Wait()
}

type pipeline[T any] struct {
	source Source[T]
	sink   Sink[T]
}

// Process processes the pipeline.
func (c pipeline[T]) Process(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, bufSize)
	g.Go(func() error {
		defer close(ch)
		return c.source.Source(ctx, ch)
	})
	g.Go(func() error {
		return c.sink.Sink(ctx, ch)
	})
	return g.Wait()
}

// Compose composes a source and a processor.
func Compose[T any](s Source[T], p Processor[T]) Source[T] {
	return source[T]{s, p}
}

// Connect connects a source and a sink.
func Connect[T any](src Source[T], snk Sink[T]) Pipeline {
	return pipeline[T]{src, snk}
}
