package model

import (
	"context"
	"fmt"
	"sync"

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
	"github.com/sboehler/knut/lib/syntax/parser"
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

func FromStream(ctx context.Context, reg *registry.Registry, ch <-chan parser.Result) <-chan any {
	out := make(chan any)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for res := range ch {
			if res.Err != nil {
				cpr.Push[any](ctx, out, res.Err)
				continue
			}
			wg.Add(1)
			res := res
			go func() {
				defer wg.Done()
				for _, d := range res.File.Directives {
					m, err := Create(reg, d.Directive)
					if err != nil {
						cpr.Push[any](ctx, out, err)
					} else if m != nil {
						cpr.Push[any](ctx, out, m)
					}
				}
			}()
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func Create(reg *registry.Registry, w any) (Directive, error) {
	switch d := w.(type) {
	case syntax.Transaction:
		return transaction.Create(reg, &d)
	case syntax.Open:
		return open.Create(reg, &d)
	case syntax.Close:
		return cls.Create(reg, &d)
	case syntax.Assertion:
		return assertion.Create(reg, &d)
	case syntax.Price:
		return price.Create(reg, &d)
	case syntax.Include:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown directive: %T", w)
}
