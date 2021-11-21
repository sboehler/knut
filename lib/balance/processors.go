package balance

import (
	"fmt"

	"github.com/sboehler/knut/lib/ledger"
)

// Processor processes the balance and the ledger day.
type Processor interface {
	Process(b *Balance, d *ledger.Day) error
}

// AccountOpener keeps track of open accounts.
type AccountOpener struct{}

var _ Processor = (*AccountOpener)(nil)

// Process implements Processor.
func (a AccountOpener) Process(b *Balance, d *ledger.Day) error {
	for _, o := range d.Openings {
		if err := b.Accounts.Open(o.Account); err != nil {
			return err
		}
	}
	return nil
}

// TransactionBooker books transaction amounts.
type TransactionBooker struct{}

var _ Processor = (*TransactionBooker)(nil)

// Process implements Processor.
func (tb TransactionBooker) Process(b *Balance, d *ledger.Day) error {
	// book journal transaction amounts
	for _, t := range d.Transactions {
		if err := b.bookAmount(t); err != nil {
			return err
		}
	}
	return nil
}

// ValueBooker books amounts for value directives.
type ValueBooker struct{}

var _ Processor = (*ValueBooker)(nil)

// Process implements Processor.
func (tb ValueBooker) Process(b *Balance, d *ledger.Day) error {
	for _, v := range d.Values {
		var (
			t   ledger.Transaction
			err error
		)
		if t, err = tb.processValue(b, v); err != nil {
			return err
		}
		if err = b.bookAmount(t); err != nil {
			return err
		}
		d.Transactions = append(d.Transactions, t)
	}
	d.Values = nil
	return nil
}

func (tb ValueBooker) processValue(b *Balance, v ledger.Value) (ledger.Transaction, error) {
	if !b.Accounts.IsOpen(v.Account) {
		return ledger.Transaction{}, Error{v, "account is not open"}
	}
	valAcc, err := b.valuationAccountFor(v.Account)
	if err != nil {
		return ledger.Transaction{}, err
	}
	var pos = CommodityAccount{v.Account, v.Commodity}
	return ledger.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v", pos),
		Tags:        nil,
		Postings: []ledger.Posting{
			ledger.NewPosting(valAcc, v.Account, pos.Commodity, v.Amount.Sub(b.Amounts[pos])),
		},
	}, nil
}

// Asserter keeps track of open accounts.
type Asserter struct{}

var _ Processor = (*Asserter)(nil)

// Process implements Processor.
func (as Asserter) Process(b *Balance, d *ledger.Day) error {
	for _, a := range d.Assertions {
		if err := as.processBalanceAssertion(b, a); err != nil {
			return err
		}
	}
	return nil
}

func (as Asserter) processBalanceAssertion(b *Balance, a ledger.Assertion) error {
	if !b.Accounts.IsOpen(a.Account) {
		return Error{a, "account is not open"}
	}
	var pos = CommodityAccount{a.Account, a.Commodity}
	va, ok := b.Amounts[pos]
	if !ok || !va.Equal(a.Amount) {
		return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, pos.Commodity)}
	}
	return nil
}
