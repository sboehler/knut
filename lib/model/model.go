package model

import (
	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/assertion"
	"github.com/sboehler/knut/lib/model/close"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/open"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/price"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
)

type Commodity = commodity.Commodity
type AccountType = account.Type
type Account = account.Account
type Posting = posting.Posting
type Transaction = transaction.Transaction
type Open = open.Open
type Close = close.Close
type Price = price.Price
type Assertion = assertion.Assertion

type Registry = registry.Registry

type Directive any

var (
	_ Directive = (*Assertion)(nil)
	_ Directive = (*Close)(nil)
	_ Directive = (*Open)(nil)
	_ Directive = (*Price)(nil)
	_ Directive = (*Transaction)(nil)
)
