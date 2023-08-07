package parser

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax/directives"
	"golang.org/x/sync/errgroup"
)

func Parse(file string) (<-chan directives.File, func(context.Context) error) {
	return cpr.Produce(func(ctx context.Context, ch chan<- directives.File) error {
		wg, ctx := errgroup.WithContext(ctx)
		wg.Go(func() error {
			res, err := parseRec(ctx, wg, ch, file)
			if err != nil {
				return err
			}
			return cpr.Push(ctx, ch, res)
		})
		return wg.Wait()
	})
}

type Result struct {
	File directives.File
	Err  error
}

func parseRec(ctx context.Context, wg *errgroup.Group, resCh chan<- directives.File, file string) (directives.File, error) {
	text, err := os.ReadFile(file)
	if err != nil {
		return directives.File{}, err
	}
	p := New(string(text), file)
	if err := p.Advance(); err != nil {
		return directives.File{}, err
	}
	p.callback = func(d directives.Directive) {
		if inc, ok := d.Directive.(directives.Include); ok {
			file := path.Join(filepath.Dir(file), inc.IncludePath.Content.Extract())
			wg.Go(func() error {
				res, err := parseRec(ctx, wg, resCh, file)
				if err != nil {
					return err
				}
				return cpr.Push(ctx, resCh, res)
			})
		}
	}
	return p.ParseFile()
}
