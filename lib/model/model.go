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

func FromStream(ctx context.Context, reg *registry.Registry, ch <-chan any) <-chan any {
	res := make(chan any)
	go func() {
		defer close(res)
		for a := range ch {
			if d, ok := a.(syntax.Directive); ok {
				m, err := Create(reg, d)
				if err != nil {
					cpr.Push[any](ctx, res, a)
				} else {
					cpr.Push[any](ctx, res, m)
				}
			} else {
				cpr.Push(ctx, res, a)
			}
		}
	}()
	return res
}

func Create(reg *registry.Registry, w syntax.Directive) (Directive, error) {
	switch d := w.Directive.(type) {
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
	}
	return nil, fmt.Errorf("unknown directive: %v", w)
}
