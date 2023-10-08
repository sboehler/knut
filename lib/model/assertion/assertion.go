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
	Src      *syntax.Assertion
	Date     time.Time
	Balances []Balance
}

type Balance struct {
	Src       *syntax.Balance
	Account   *account.Account
	Quantity  decimal.Decimal
	Commodity *commodity.Commodity
}

func Create(reg *registry.Registry, a *syntax.Assertion) (*Assertion, error) {
	date, err := a.Date.Parse()
	if err != nil {
		return nil, err
	}
	balances := make([]Balance, 0, len(a.Balances))
	for _, bal := range a.Balances {
		account, err := reg.Accounts().Create(bal.Account)
		if err != nil {
			return nil, err
		}
		quantity, err := bal.Quantity.Parse()
		if err != nil {
			return nil, err
		}
		commodity, err := reg.Commodities().Create(bal.Commodity)
		if err != nil {
			return nil, err
		}
		balances = append(balances, Balance{
			Src:       &bal,
			Account:   account,
			Quantity:  quantity,
			Commodity: commodity,
		})

	}
	return &Assertion{
		Src:      a,
		Date:     date,
		Balances: balances,
	}, nil
}
