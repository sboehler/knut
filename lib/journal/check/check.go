package check

import (
	"fmt"
	"strings"

	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/printer"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/assertion"
	"golang.org/x/exp/slices"
)

// Error is a processing error, with a reference to a directive with
// a source location.
type Error struct {
	Directive model.Directive
	Msg       string
}

func (be Error) Error() string {
	var s strings.Builder
	s.WriteString(be.Msg)
	s.WriteRune('\n')
	s.WriteRune('\n')
	p := printer.New(&s)
	p.PrintDirectiveLn(be.Directive)
	return s.String()
}

type Checker struct {
	Write bool

	quantities amounts.Amounts
	accounts   set.Set[*model.Account]
	assertions []*model.Assertion
}

func (ch *Checker) Assertions() []*model.Assertion {
	return ch.assertions
}

func (ch *Checker) open(o *model.Open) error {
	if ch.accounts.Has(o.Account) {
		return Error{Directive: o, Msg: "account is already open"}
	}
	ch.accounts.Add(o.Account)
	return nil
}

func (ch *Checker) posting(t *model.Transaction, p *model.Posting) error {
	if !ch.accounts.Has(p.Account) {
		return Error{Directive: t, Msg: fmt.Sprintf("account %s is not open", p.Account)}
	}
	if p.Account.IsAL() {
		ch.quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
	}
	return nil
}

func (ch *Checker) balance(a *model.Assertion, bal *model.Balance) error {
	if !ch.accounts.Has(bal.Account) {
		return Error{Directive: a, Msg: "account is not open"}
	}
	position := amounts.AccountCommodityKey(bal.Account, bal.Commodity)
	if qty, ok := ch.quantities[position]; !ok || !qty.Equal(bal.Quantity) {
		return Error{Directive: a, Msg: fmt.Sprintf("failed assertion: %s has position: %s %s", position.Account.Name(), qty, position.Commodity.Name())}
	}
	return nil
}

func (ch *Checker) close(c *model.Close) error {
	for pos, amount := range ch.quantities {
		if pos.Account != c.Account {
			continue
		}
		if !amount.IsZero() {
			return Error{Directive: c, Msg: fmt.Sprintf("account has nonzero position: %s %s", amount, pos.Commodity.Name())}
		}
		delete(ch.quantities, pos)
	}
	if !ch.accounts.Has(c.Account) {
		return Error{Directive: c, Msg: "account is not open"}
	}
	ch.accounts.Remove(c.Account)
	return nil
}

func (ch *Checker) dayEnd(d *journal.Day) error {
	if len(ch.quantities) == 0 {
		return nil
	}
	bal := make([]model.Balance, 0, len(ch.quantities))
	for pos, qty := range ch.quantities {
		bal = append(bal, model.Balance{
			Account:   pos.Account,
			Quantity:  qty,
			Commodity: pos.Commodity,
		})
	}
	slices.SortFunc(bal, assertion.CompareBalance)
	ch.assertions = append(ch.assertions, &model.Assertion{
		Date:     d.Date,
		Balances: bal,
	})
	return nil
}

func (ch *Checker) Check(create bool) *journal.Processor {
	ch.quantities = make(amounts.Amounts)
	ch.accounts = set.New[*model.Account]()
	ch.assertions = nil

	var dayEnd func(*journal.Day) error
	if create {
		dayEnd = ch.dayEnd
	}

	return &journal.Processor{
		Open:    ch.open,
		Posting: ch.posting,
		Balance: ch.balance,
		Close:   ch.close,
		DayEnd:  dayEnd,
	}
}

// Checker checks the journal (with default options).
func Check() *journal.Processor {
	var checker Checker
	return checker.Check(false)
}
