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
type Processor[T any, U any] interface {
	Process(context.Context, <-chan T, chan<- U) error
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
func (p *Producer[T]) Source(ctx context.Context, outCh chan<- T) error {
	for _, i := range p.Items {
		if err := Push(ctx, outCh, i); err != nil {
			return err
		}
	}
	return nil
}

// RunTestEngine runs the processor in a test engine.
func RunTestEngine[T any, U any](ctx context.Context, ps Processor[T, U], ts ...T) ([]U, error) {
	sink := new(Collector[U])
	ppl := Connect[U](Compose[T](&Producer[T]{ts}, ps), sink)
	if err := ppl.Process(ctx); err != nil {
		return nil, err
	}
	return sink.Result, nil
}

// Pipeline represents a computation.
type Pipeline interface {
	Process(context.Context) error
}

type source[T any, U any] struct {
	source Source[T]
	proc   Processor[T, U]
}

func (src source[T, U]) Source(ctx context.Context, oCh chan<- U) error {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, bufSize)
	g.Go(func() error {
		defer close(ch)
		return src.source.Source(ctx, ch)
	})
	g.Go(func() error {
		return src.proc.Process(ctx, ch, oCh)
	})
	return g.Wait()
}

type pipeline[T any] struct {
	source Source[T]
	sink   Sink[T]
}

// Process processes the pipeline.
func (ppl pipeline[T]) Process(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan T, bufSize)
	g.Go(func() error {
		defer close(ch)
		return ppl.source.Source(ctx, ch)
	})
	g.Go(func() error {
		return ppl.sink.Sink(ctx, ch)
	})
	return g.Wait()
}

// Compose composes a source and a processor.
func Compose[T any, U any](s Source[T], p Processor[T, U]) Source[U] {
	return source[T, U]{s, p}
}

// Connect connects a source and a sink.
func Connect[T any](src Source[T], snk Sink[T]) Pipeline {
	return pipeline[T]{src, snk}
}
