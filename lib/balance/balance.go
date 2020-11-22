// Copyright 2020 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package balance

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/amount"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/accounts"
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/prices"

	"github.com/shopspring/decimal"
)

// Balance represents a balance for accounts at the given date.
type Balance struct {
	Date             time.Time
	Positions        map[model.CommodityAccount]amount.Amount
	Account          map[*accounts.Account]bool
	Prices           prices.Prices
	Valuations       []*commodities.Commodity
	NormalizedPrices []prices.NormalizedPrices
}

// New creates a new balance.
func New(valuations []*commodities.Commodity) *Balance {
	return &Balance{
		Positions:        map[model.CommodityAccount]amount.Amount{},
		Account:          map[*accounts.Account]bool{accounts.ValuationAccount(): true},
		Prices:           prices.Prices{},
		Valuations:       valuations,
		NormalizedPrices: make([]prices.NormalizedPrices, len(valuations)),
	}
}

// Copy deeply copies the balance
func (b *Balance) Copy() *Balance {
	nb := &Balance{
		Positions: map[model.CommodityAccount]amount.Amount{},
		Account:   map[*accounts.Account]bool{accounts.ValuationAccount(): true},
		Prices:    b.Prices.Copy(),
		// the following fields are immutable
		Valuations:       b.Valuations,
		NormalizedPrices: b.NormalizedPrices,
	}
	nb.Date = b.Date
	for pos, val := range b.Positions {
		nb.Positions[pos] = val
	}
	for acc := range b.Account {
		nb.Account[acc] = true
	}
	return nb
}

// Minus mutably subtracts the given balance from the receiver.
func (b *Balance) Minus(bo *Balance) {
	for pos, va := range bo.Positions {
		b.Positions[pos] = b.Positions[pos].Minus(va)
	}
}

// Update updates the balance with the given step
func (b *Balance) Update(step *ledger.Step) error {

	// update date
	b.Date = step.Date

	// update prices
	for _, p := range step.Prices {
		b.Prices.Insert(p)
	}

	// update normalized prices
	b.NormalizedPrices = make([]prices.NormalizedPrices, 0, len(b.NormalizedPrices))
	for _, c := range b.Valuations {
		b.NormalizedPrices = append(b.NormalizedPrices, b.Prices.Normalize(c))
	}

	// open accounts
	for _, o := range step.Openings {
		if _, isOpen := b.Account[o.Account]; isOpen {
			return fmt.Errorf("Account %v is already open", o)
		}
		b.Account[o.Account] = true
	}

	// valuate transactions
	for _, t := range step.Transactions {
		if err := b.valuateTransaction(t); err != nil {
			return err
		}
	}

	// compute and append valuation transactions
	valTrx, err := b.computeValuationTransactions()
	if err != nil {
		return err
	}
	step.Transactions = append(step.Transactions, valTrx...)

	// book transactions
	for _, t := range step.Transactions {
		if err := b.bookTransaction(t); err != nil {
			return err
		}
	}

	// process balance assertions
	for _, a := range step.Assertions {
		if err := b.processBalanceAssertion(a); err != nil {
			return err
		}
	}

	// close accounts
	for _, c := range step.Closings {
		if _, isOpen := b.Account[c.Account]; !isOpen {
			return fmt.Errorf("Account %v is not open", c)
		}
		for pos, amount := range b.Positions {
			if pos.Account() == c.Account && !amount.Amount().IsZero() {
				return fmt.Errorf("Account %v has nonzero position", pos)
			}
		}
		delete(b.Account, c.Account)
	}
	return nil
}

func (b *Balance) bookTransaction(t *model.Transaction) error {
	for _, posting := range t.Postings {
		if _, isOpen := b.Account[posting.Credit]; !isOpen {
			return fmt.Errorf("Account %v is not open", t)
		}
		if _, isOpen := b.Account[posting.Debit]; !isOpen {
			return fmt.Errorf("Account %v is not open", t)
		}
		crPos := model.NewCommodityAccount(posting.Credit, posting.Commodity)
		drPos := model.NewCommodityAccount(posting.Debit, posting.Commodity)
		b.Positions[crPos] = b.Positions[crPos].Minus(posting.Amount)
		b.Positions[drPos] = b.Positions[drPos].Plus(posting.Amount)
	}
	return nil
}

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (b *Balance) computeValuationTransactions() ([]*model.Transaction, error) {
	result := []*model.Transaction{}
	for pos, va := range b.Positions {
		at := pos.Account().Type()
		if at != accounts.ASSETS && at != accounts.LIABILITIES {
			continue
		}
		diffs, nonzero, err := b.computeValuationDiff(pos, va)
		if err != nil {
			return nil, err
		}
		if nonzero {
			// create a transaction to adjust the valuation
			result = append(result, &model.Transaction{
				Directive:   model.NewDirective(model.Range{}, b.Date),
				Description: fmt.Sprintf("Valuation adjustment for %v", pos),
				Tags:        nil,
				Postings: []*model.Posting{
					{
						Amount:    amount.New(decimal.Zero, diffs),
						Credit:    accounts.ValuationAccount(),
						Debit:     pos.Account(),
						Commodity: pos.Commodity(),
					},
				},
			})
		}
	}
	return result, nil
}

func (b *Balance) computeValuationDiff(pos model.CommodityAccount, va amount.Amount) ([]decimal.Decimal, bool, error) {
	diffs := make([]decimal.Decimal, len(b.NormalizedPrices))
	nonzero := false
	for i, np := range b.NormalizedPrices {
		v1 := va.Valuation(i)
		v2, err := np.Valuate(pos.Commodity(), va.Amount())
		if err != nil {
			return nil, false, fmt.Errorf("Should not hapen - no valuation found")
		}
		if !v1.Equal(v2) {
			diffs[i] = v2.Sub(v1)
			nonzero = true
		}
	}
	return diffs, nonzero, nil
}

func (b *Balance) valuateTransaction(t *model.Transaction) error {
	for _, posting := range t.Postings {
		valuations := make([]decimal.Decimal, 0, len(b.NormalizedPrices))
		for _, np := range b.NormalizedPrices {
			value, err := np.Valuate(posting.Commodity, posting.Amount.Amount())
			if err != nil {
				return fmt.Errorf("No price found for %v in %v", posting.Commodity, np)
			}
			valuations = append(valuations, value)
		}
		posting.Amount = amount.New(posting.Amount.Amount(), valuations)
	}
	return nil
}

func (b *Balance) processBalanceAssertion(a *model.Assertion) error {
	pos := model.NewCommodityAccount(a.Account, a.Commodity)
	va, ok := b.Positions[pos]
	if !ok || !va.Amount().Equal(a.Amount) {
		return fmt.Errorf("Balance assertion failed: Expected %v, got %v", a, va)
	}
	return nil
}

// GetPositions returns the positions for the given valuation index.
// An index of nil returns the raw counts.
func (b *Balance) GetPositions(valuation *int) map[model.CommodityAccount]decimal.Decimal {
	res := make(map[model.CommodityAccount]decimal.Decimal, len(b.Positions))
	for pos, amt := range b.Positions {
		if valuation == nil {
			res[pos] = amt.Amount()
		} else {
			res[pos] = amt.Valuation(*valuation)
		}
	}
	return res
}

// Options has options for processing a ledger

// Diffs creates the difference balances for the given
// slice of balances. The returned slice is one element smaller
// than the input slice. The balances are mutated.
func Diffs(bals []*Balance) []*Balance {
	for i := len(bals) - 1; i > 0; i-- {
		bals[i].Minus(bals[i-1])
	}
	return bals[1:]
}
