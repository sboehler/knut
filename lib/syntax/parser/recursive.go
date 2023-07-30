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

func Parse(ctx context.Context, file string) <-chan any {
	resCh := make(chan any, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := parseRec(ctx, &wg, resCh, file)
		if err != nil && ctx.Err() == nil {
			cpr.Push[any](ctx, resCh, err)
		}
	}()

	// Parse and eventually close input channel
	go func() {
		defer close(resCh)
		wg.Wait()
	}()
	return resCh
}

func parseRec(ctx context.Context, wg *sync.WaitGroup, resCh chan<- any, file string) error {
	text, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	p := New(string(text), file)
	if err := p.Advance(); err != nil {
		return err
	}
	p.callback = func(d syntax.Directive) error {
		if inc, ok := d.Directive.(syntax.Include); ok {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p := path.Join(filepath.Dir(file), inc.Path.Content.Extract())
				err := parseRec(ctx, wg, resCh, p)
				if err != nil && ctx.Err() == nil {
					cpr.Push[any](ctx, resCh, err)
				}
			}()
			return nil
		}
		return cpr.Push[any](ctx, resCh, d.Directive)
	}
	_, err = p.ParseFile()
	return err
}
