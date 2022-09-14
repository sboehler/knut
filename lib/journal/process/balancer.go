package process

import (
	"context"
	"fmt"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Balancer processes ASTs.
type Balancer struct {
	Context journal.Context
}

// Process processes days.
func (pr *Balancer) Process(ctx context.Context, inCh <-chan *ast.Day, outCh chan<- *ast.Day) error {
	amounts := make(amounts.Amounts)
	accounts := make(accounts)

	return cpr.Consume(ctx, inCh, func(d *ast.Day) error {
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

func (pr *Balancer) processOpenings(ctx context.Context, accounts accounts, d *ast.Day) error {
	for _, o := range d.Openings {
		if err := accounts.Open(o.Account); err != nil {
			return err
		}
	}
	return nil
}

func (pr *Balancer) processTransactions(ctx context.Context, accounts accounts, amts amounts.Amounts, d *ast.Day) error {
	for _, t := range d.Transactions {
		for _, p := range t.Postings() {
			if !accounts.IsOpen(p.Credit) {
				return Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
			}
			if !accounts.IsOpen(p.Debit) {
				return Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
			}
			amts.Add(amounts.AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
			amts.Add(amounts.AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
		}
	}
	return nil
}

func (pr *Balancer) processValues(ctx context.Context, accounts accounts, amts amounts.Amounts, d *ast.Day) error {
	for _, v := range d.Values {
		if !accounts.IsOpen(v.Account) {
			return Error{v, "account is not open"}
		}
		valAcc := pr.Context.ValuationAccountFor(v.Account)
		p := ast.PostingWithTargets(valAcc, v.Account, v.Commodity, v.Amount.Sub(amts.Amount(amounts.AccountCommodityKey(v.Account, v.Commodity))), []*journal.Commodity{v.Commodity})
		amts.Add(amounts.AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
		amts.Add(amounts.AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
		d.Transactions = append(d.Transactions, ast.TransactionBuilder{
			Date:        v.Date,
			Description: fmt.Sprintf("Valuation adjustment for %v in %v", v.Commodity, v.Account),
			Postings:    []ast.Posting{p},
		}.Build())
	}
	compare.Sort(d.Transactions, ast.CompareTransactions)
	return nil
}

func (pr *Balancer) processAssertions(ctx context.Context, accounts accounts, amts amounts.Amounts, d *ast.Day) error {
	for _, a := range d.Assertions {
		if !accounts.IsOpen(a.Account) {
			return Error{a, "account is not open"}
		}
		position := amounts.AccountCommodityKey(a.Account, a.Commodity)
		if va, ok := amts[position]; !ok || !va.Equal(a.Amount) {
			return Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account.Name(), va, position.Commodity.Name())}
		}
	}
	return nil
}

func (pr *Balancer) processClosings(ctx context.Context, accounts accounts, amounts amounts.Amounts, d *ast.Day) error {
	for _, c := range d.Closings {
		for pos, amount := range amounts {
			if pos.Account != c.Account {
				continue
			}
			if !amount.IsZero() {
				return Error{c, "account has nonzero position"}
			}
			delete(amounts, pos)
		}
		if err := accounts.Close(c.Account); err != nil {
			return err
		}
	}
	return nil
}

// accounts keeps track of open accounts.
type accounts map[*journal.Account]bool

// Open opens an account.
func (oa accounts) Open(a *journal.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa accounts) Close(a *journal.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa accounts) IsOpen(a *journal.Account) bool {
	return oa[a] || a.Type() == journal.EQUITY
}
