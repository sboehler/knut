package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
)

// Balancer processes ASTs.
type Balancer struct {
	Context journal.Context
}

// Process processes days.
func (pr *Balancer) Process(ctx context.Context, inCh <-chan *journal.Day, outCh chan<- *journal.Day) error {
	amounts := make(journal.Amounts)
	accounts := make(accounts)

	return cpr.Consume(ctx, inCh, func(d *journal.Day) error {
		if err := pr.processOpenings(ctx, accounts, d); err != nil {
			return err
		}
		if err := pr.processTransactions(ctx, accounts, amounts, d); err != nil {
			return err
		}
		if err := pr.processValues(ctx, accounts, amounts, d); err != nil {
			return err
		}
		if err := pr.processAssertions(ctx, accounts, amounts, d); err != nil {
			return err
		}
		if err := pr.processClosings(ctx, accounts, amounts, d); err != nil {
			return err
		}
		d.Amounts = amounts.Clone()
		return cpr.Push(ctx, outCh, d)
	})
}

func (pr *Balancer) processOpenings(ctx context.Context, accounts accounts, d *journal.Day) error {
	for _, o := range d.Openings {
		if ok := accounts.Open(o.Account); !ok {
			return Error{o, "account is already open"}
		}
	}
	return nil
}

func (pr *Balancer) processTransactions(ctx context.Context, accounts accounts, amts journal.Amounts, d *journal.Day) error {
	for _, t := range d.Transactions {
		for _, p := range t.Postings {
			if !accounts.IsOpen(p.Credit) {
				return Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
			}
			if !accounts.IsOpen(p.Debit) {
				return Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
			}
			amts.Add(journal.AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
			amts.Add(journal.AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
		}
	}
	return nil
}

func (pr *Balancer) processValues(ctx context.Context, accounts accounts, amts journal.Amounts, d *journal.Day) error {
	for _, v := range d.Values {
		if !accounts.IsOpen(v.Account) {
			return Error{v, "account is not open"}
		}
		valAcc := pr.Context.ValuationAccountFor(v.Account)
		p := journal.PostingWithTargets(valAcc, v.Account, v.Commodity, v.Amount.Sub(amts.Amount(journal.AccountCommodityKey(v.Account, v.Commodity))), []*journal.Commodity{v.Commodity})
		d.Transactions = append(d.Transactions, journal.TransactionBuilder{
			Date:        v.Date,
			Description: fmt.Sprintf("Valuation adjustment for %s in %s", v.Commodity.Name(), v.Account.Name()),
			Postings:    []journal.Posting{p},
		}.Build())
		amts.Add(journal.AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
		amts.Add(journal.AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
	}
	compare.Sort(d.Transactions, journal.CompareTransactions)
	return nil
}

func (pr *Balancer) processAssertions(ctx context.Context, accounts accounts, amts journal.Amounts, d *journal.Day) error {
	for _, a := range d.Assertions {
		if !accounts.IsOpen(a.Account) {
			return Error{a, "account is not open"}
		}
		position := journal.AccountCommodityKey(a.Account, a.Commodity)
		if va, ok := amts[position]; !ok || !va.Equal(a.Amount) {
			return Error{a, fmt.Sprintf("account has position: %s %s", va, position.Commodity.Name())}
		}
	}
	return nil
}

func (pr *Balancer) processClosings(ctx context.Context, accounts accounts, amounts journal.Amounts, d *journal.Day) error {
	for _, c := range d.Closings {
		for pos, amount := range amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() {
				return Error{c, fmt.Sprintf("account has nonzero position: %s %s", amount, pos.Commodity.Name())}
			}
			delete(amounts, pos)
		}
		if ok := accounts.Close(c.Account); !ok {
			return Error{c, "account is not open"}
		}
	}
	return nil
}

// accounts keeps track of open accounts.
type accounts map[*journal.Account]struct{}

// Open opens an account.
func (oa accounts) Open(a *journal.Account) bool {
	if _, open := oa[a]; open {
		return false
	}
	oa[a] = struct{}{}
	return true
}

// Close closes an account.
func (oa accounts) Close(a *journal.Account) bool {
	if _, open := oa[a]; !open {
		return false
	}
	delete(oa, a)
	return true
}

// IsOpen returns whether an account is open.
func (oa accounts) IsOpen(a *journal.Account) bool {
	_, open := oa[a]
	return open
}
