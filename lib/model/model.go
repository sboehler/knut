package model

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/assertion"
	cls "github.com/sboehler/knut/lib/model/close"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/open"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/price"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/sboehler/knut/lib/syntax"
	"golang.org/x/sync/errgroup"
)

type Commodity = commodity.Commodity
type AccountType = account.Type
type Account = account.Account
type Posting = posting.Posting
type Transaction = transaction.Transaction
type Open = open.Open
type Close = cls.Close
type Price = price.Price
type Assertion = assertion.Assertion

type Registry = registry.Registry

type Directive any

var (
	_ Directive = (*assertion.Assertion)(nil)
	_ Directive = (*cls.Close)(nil)
	_ Directive = (*open.Open)(nil)
	_ Directive = (*price.Price)(nil)
	_ Directive = (*transaction.Transaction)(nil)
)

type Result struct {
	Err        error
	Directives []any
}

func FromStream(reg *registry.Registry, inCh <-chan syntax.File) (<-chan []any, func(context.Context) error) {
	return cpr.Produce2(func(ctx context.Context, ch chan<- []any) error {
		var wg errgroup.Group
		cpr.Consume(ctx, inCh, func(input syntax.File) error {
			wg.Go(func() error {
				var ds []any
				for _, d := range input.Directives {
					m, err := Create(reg, d.Directive)
					if err != nil {
						return err
					}
					ds = append(ds, m...)
				}
				return cpr.Push(ctx, ch, ds)
			})
			return nil
		})
		return wg.Wait()
	})
}

func Create(reg *registry.Registry, w any) ([]any, error) {
	switch d := w.(type) {
	case syntax.Transaction:
		ts, err := transaction.Create(reg, &d)
		if err != nil {
			return nil, err
		}
		var res []any
		for _, t := range ts {
			res = append(res, t)
		}
		return res, nil
	case syntax.Open:
		o, err := open.Create(reg, &d)
		if err != nil {
			return nil, err
		}
		return []any{o}, nil
	case syntax.Close:
		o, err := cls.Create(reg, &d)
		if err != nil {
			return nil, err
		}
		return []any{o}, nil
	case syntax.Assertion:
		o, err := assertion.Create(reg, &d)
		if err != nil {
			return nil, err
		}
		return []any{o}, nil
	case syntax.Price:
		o, err := price.Create(reg, &d)
		if err != nil {
			return nil, err
		}
		return []any{o}, nil
	case syntax.Include:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown directive: %T", w)
}
