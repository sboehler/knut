package syntax

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"text/scanner"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/syntax/directives"
	"github.com/sboehler/knut/lib/syntax/parser"
	"github.com/sboehler/knut/lib/syntax/printer"
	"golang.org/x/sync/errgroup"
)

type Commodity = directives.Commodity

type Account = directives.Account

type Date = directives.Date

type Decimal = directives.Decimal

type QuotedString = directives.QuotedString

type Booking = directives.Booking

type Performance = directives.Performance

type Interval = directives.Interval

type Directive = directives.Directive

type File = directives.File

type Accrual = directives.Accrual

type Addons = directives.Addons

type Transaction = directives.Transaction

type Open = directives.Open

type Close = directives.Close

type Assertion = directives.Assertion

type Balance = directives.Balance

type Price = directives.Price

type Include = directives.Include

type Range = directives.Range

type Location = directives.Location

var _ error = Error{}

type Error = directives.Error

type Parser = parser.Parser

type Scanner = scanner.Scanner

func ParseFile(file string) (directives.File, error) {
	text, err := os.ReadFile(file)
	if err != nil {
		return directives.File{}, err
	}
	p := parser.New(string(text), file)
	if err := p.Advance(); err != nil {
		return directives.File{}, err
	}
	return p.ParseFile()
}

func ParseFileRecursively(file string) (<-chan directives.File, func(context.Context) error) {
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
	p := parser.New(string(text), file)
	if err := p.Advance(); err != nil {
		return directives.File{}, err
	}
	p.Callback = func(d directives.Directive) {
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

func FormatFile(w io.Writer, f directives.File) error {
	p := printer.New(w)
	return p.Format(f)
}

func PrintFile(w io.Writer, f directives.File) error {
	p := printer.New(w)
	_, err := p.PrintFile(f)
	return err
}
