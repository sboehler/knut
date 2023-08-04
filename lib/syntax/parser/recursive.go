package parser

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax"
	"golang.org/x/sync/errgroup"
)

func Parse(file string) (<-chan syntax.File, func(context.Context) error) {
	return cpr.Produce2(func(ctx context.Context, ch chan<- syntax.File) error {
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
	File syntax.File
	Err  error
}

func parseRec(ctx context.Context, wg *errgroup.Group, resCh chan<- syntax.File, file string) (syntax.File, error) {
	text, err := os.ReadFile(file)
	if err != nil {
		return syntax.File{}, err
	}
	p := New(string(text), file)
	if err := p.Advance(); err != nil {
		return syntax.File{}, err
	}
	p.callback = func(d syntax.Directive) {
		if inc, ok := d.Directive.(syntax.Include); ok {
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
