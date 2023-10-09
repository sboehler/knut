package check

import (
	"fmt"

	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
)

// Balance balances the journal.
func Check() *journal.Processor {
	quantities := make(amounts.Amounts)
	accounts := set.New[*model.Account]()

	return &journal.Processor{

		Open: func(o *model.Open) error {
			if accounts.Has(o.Account) {
				return journal.Error{Directive: o, Msg: "account is already open"}
			}
			accounts.Add(o.Account)
			return nil
		},

		Posting: func(t *model.Transaction, p *model.Posting) error {
			if !accounts.Has(p.Account) {
				return journal.Error{Directive: t, Msg: fmt.Sprintf("account %s is not open", p.Account)}
			}
			if p.Account.IsAL() {
				quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
			}
			return nil
		},

		Balance: func(a *model.Balance) error {
			if !accounts.Has(a.Account) {
				return journal.Error{Directive: a, Msg: "account is not open"}
			}
			position := amounts.AccountCommodityKey(a.Account, a.Commodity)
			if qty, ok := quantities[position]; !ok || !qty.Equal(a.Quantity) {
				return journal.Error{Directive: a, Msg: fmt.Sprintf("failed assertion: account has position: %s %s", qty, position.Commodity.Name())}
			}
			return nil
		},

		Close: func(c *model.Close) error {
			for pos, amount := range quantities {
				if pos.Account != c.Account {
					continue
				}
				if !amount.IsZero() {
					return journal.Error{Directive: c, Msg: fmt.Sprintf("account has nonzero position: %s %s", amount, pos.Commodity.Name())}
				}
				delete(quantities, pos)
			}
			if !accounts.Has(c.Account) {
				return journal.Error{Directive: c, Msg: "account is not open"}
			}
			accounts.Remove(c.Account)
			return nil
		},
	}
}
