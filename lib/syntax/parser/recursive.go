package parser

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/sourcegraph/conc"
)

func Parse(ctx context.Context, file string) <-chan Result {
	return cpr.Produce(func(wg *conc.WaitGroup, ch chan<- Result) {
		res := parseRec(ctx, wg, ch, file)
		cpr.Push(ctx, ch, res)
	})
}

type Result struct {
	File syntax.File
	Err  error
}

func parseRec(ctx context.Context, wg *conc.WaitGroup, resCh chan<- Result, file string) Result {
	text, err := os.ReadFile(file)
	if err != nil {
		return Result{Err: err}
	}
	p := New(string(text), file)
	if err := p.Advance(); err != nil {
		return Result{Err: err}
	}
	p.callback = func(d syntax.Directive) {
		if inc, ok := d.Directive.(syntax.Include); ok {
			file := path.Join(filepath.Dir(file), inc.IncludePath.Content.Extract())
			wg.Go(func() {
				res := parseRec(ctx, wg, resCh, file)
				cpr.Push(ctx, resCh, res)
			})
		}
	}
	f, err := p.ParseFile()
	return Result{File: f, Err: err}
}
