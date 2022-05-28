package amounts

import (
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Account   *journal.Account
	Commodity *journal.Commodity
}

func CommodityKey(c *journal.Commodity) Key {
	return Key{Commodity: c}
}

func AccountKey(a *journal.Account) Key {
	return Key{Account: a}
}

func AccountCommodityKey(a *journal.Account, c *journal.Commodity) Key {
	return Key{Account: a, Commodity: c}
}

// Less establishes a partial ordering of keys.
func (p Key) Less(p1 Key) bool {
	if p.Account.Type() != p1.Account.Type() {
		return p.Account.Type() < p1.Account.Type()
	}
	if p.Account.String() != p1.Account.String() {
		return p.Account.String() < p1.Account.String()
	}
	return p.Commodity.String() < p1.Commodity.String()
}

// Amounts keeps track of amounts by account and commodity.
type Amounts map[Key]decimal.Decimal

// Amount returns the amount for the given account and commodity.
func (am Amounts) Amount(a *journal.Account, c *journal.Commodity) decimal.Decimal {
	return am[Key{Account: a, Commodity: c}]
}

func (am Amounts) Add(k Key, d decimal.Decimal) {
	am[k] = am[k].Add(d)
}

// Clone clones these amounts.
func (am Amounts) Clone() Amounts {
	clone := make(Amounts)
	for ca, v := range am {
		clone[ca] = v
	}
	return clone
}

// Minus mutably subtracts.
func (am Amounts) Minus(a Amounts) Amounts {
	for ca, v := range a {
		am[ca] = am[ca].Sub(v)
	}
	return am
}
