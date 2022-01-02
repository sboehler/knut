package amounts

import (
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// CommodityAccount represents a position.
type CommodityAccount struct {
	Account   *journal.Account
	Commodity *journal.Commodity
}

// Less establishes a partial ordering of commodity accounts.
func (p CommodityAccount) Less(p1 CommodityAccount) bool {
	if p.Account.Type() != p1.Account.Type() {
		return p.Account.Type() < p1.Account.Type()
	}
	if p.Account.String() != p1.Account.String() {
		return p.Account.String() < p1.Account.String()
	}
	return p.Commodity.String() < p1.Commodity.String()
}

// Amounts keeps track of amounts by account and commodity.
type Amounts map[CommodityAccount]decimal.Decimal

// Amount returns the amount for the given account and commodity.
func (am Amounts) Amount(a *journal.Account, c *journal.Commodity) decimal.Decimal {
	return am[CommodityAccount{Account: a, Commodity: c}]
}

// Book books the given amount.
func (am Amounts) Book(cr, dr *journal.Account, a decimal.Decimal, c *journal.Commodity) {
	var (
		crPos = CommodityAccount{cr, c}
		drPos = CommodityAccount{dr, c}
	)
	am[crPos] = am[crPos].Sub(a)
	am[drPos] = am[drPos].Add(a)
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
func (am Amounts) Minus(a Amounts) {
	for ca, v := range a {
		am[ca] = am[ca].Sub(v)
	}
}
