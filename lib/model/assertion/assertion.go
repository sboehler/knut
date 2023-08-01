package assertion

import (
	"time"

	"github.com/sboehler/knut/lib/model/account"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

// Assertion represents a balance assertion.
type Assertion struct {
	Src       *syntax.Assertion
	Date      time.Time
	Account   *account.Account
	Amount    decimal.Decimal
	Commodity *commodity.Commodity
}

func Create(reg *registry.Registry, o *syntax.Assertion) (*Assertion, error) {
	account, err := reg.Accounts().Create(o.Account)
	if err != nil {
		return nil, err
	}
	date, err := o.Date.Parse()
	if err != nil {
		return nil, err
	}
	amount, err := o.Amount.Parse()
	if err != nil {
		return nil, err
	}
	commodity, err := reg.Commodities().Create(o.Commodity)
	if err != nil {
		return nil, err
	}
	return &Assertion{
		Src:       o,
		Date:      date,
		Account:   account,
		Amount:    amount,
		Commodity: commodity,
	}, nil
}
