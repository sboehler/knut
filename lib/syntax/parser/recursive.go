package parser

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax"
)

func Parse(ctx context.Context, file string) <-chan Result {
	resCh := make(chan Result)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		res := parseRec(ctx, &wg, resCh, file)
		cpr.Push(ctx, resCh, res)
	}()

	// Parse and eventually close input channel
	go func() {
		defer close(resCh)
		wg.Wait()
	}()
	return resCh
}

type Result struct {
	File syntax.File
	Err  error
}

func parseRec(ctx context.Context, wg *sync.WaitGroup, resCh chan<- Result, file string) Result {
	text, err := os.ReadFile(file)
	if err != nil {
		return Result{Err: err}
	}
	p := New(string(text), file)
	if err := p.Advance(); err != nil {
		return Result{Err: err}
	}
	p.callback = func(d syntax.Directive) error {
		if inc, ok := d.Directive.(syntax.Include); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p := path.Join(filepath.Dir(file), inc.Path.Content.Extract())
				res := parseRec(ctx, wg, resCh, p)
				cpr.Push(ctx, resCh, res)
			}()
			return nil
		}
		return nil
	}
	f, err := p.ParseFile()
	return Result{File: f, Err: err}
}
