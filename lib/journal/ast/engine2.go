package ast

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Source2 generates elements.
type Source2 interface {
	Source2(context.Context, *errgroup.Group) <-chan *Day
}

// Processor2 processes elements.
type Processor2 interface {
	Process2(context.Context, *errgroup.Group, <-chan *Day) <-chan *Day
}

// Sink2 consumes elements.
type Sink2 interface {
	Sink2(context.Context, *errgroup.Group, <-chan *Day)
}

// Engine2 processes a pipeline.
type Engine2 struct {
	Source     Source2
	Sink       Sink2
	Processors []Processor2
}

// Process processes the pipeline in the engine.
func (eng *Engine2) Process(ctx context.Context) error {
	grp, ctx := errgroup.WithContext(ctx)

	ch := eng.Source.Source2(ctx, grp)

	for _, pr := range eng.Processors {
		ch = pr.Process2(ctx, grp, ch)
	}

	eng.Sink.Sink2(ctx, grp, ch)
	return grp.Wait()
}

// Add adds a processor.
func (eng *Engine2) Add(p Processor2) {
	eng.Processors = append(eng.Processors, p)
}
