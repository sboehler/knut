package model

import (
	"time"

	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

type Commodity = commodity.Commodity
type AccountType = account.Type
type Account = account.Account
type Posting = posting.Posting
type Transaction = transaction.Transaction

type Registry = registry.Registry

// Directive is an element in a journal.
type Directive any

var (
	_ Directive = (*Assertion)(nil)
	_ Directive = (*Close)(nil)
	_ Directive = (*Open)(nil)
	_ Directive = (*Price)(nil)
	_ Directive = (*Transaction)(nil)
)

// Open represents an open command.
type Open struct {
	Src     *syntax.Open
	Date    time.Time
	Account *Account
}

// Close represents a close command.
type Close struct {
	Src     *syntax.Close
	Date    time.Time
	Account *Account
}

// Price represents a price command.
type Price struct {
	Src       *syntax.Price
	Date      time.Time
	Commodity *Commodity
	Target    *Commodity
	Price     decimal.Decimal
}

// Assertion represents a balance assertion.
type Assertion struct {
	Src       *syntax.Assertion
	Date      time.Time
	Account   *Account
	Amount    decimal.Decimal
	Commodity *Commodity
}
