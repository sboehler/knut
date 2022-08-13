package register

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Register represents a register report.
type Register struct {
	Domain journal.Filter
	Filter journal.Filter

	sections []*Section
}

// Add adds another day.
func (r *Register) Add(d *ast.Day) {
	vals := make(amounts.Amounts)
	amts := make(amounts.Amounts)
	for _, t := range d.Transactions {
		for _, b := range t.Postings() {
			if !r.Domain.MatchCommodity(b.Commodity) {
				continue
			}
			inCr := r.Domain.MatchAccount(b.Credit)
			inDr := r.Domain.MatchAccount(b.Debit)
			if inCr == inDr {
				continue
			}

			if inCr && r.Filter.MatchAccount(b.Debit) {
				ca := amounts.Key{Account: b.Debit, Commodity: b.Commodity}
				amts[ca] = amts[ca].Sub(b.Amount)
				vals[ca] = vals[ca].Sub(b.Value)

			}
			if inDr && r.Filter.MatchAccount(b.Credit) {
				ca := amounts.Key{Account: b.Credit, Commodity: b.Commodity}
				amts[ca] = amts[ca].Add(b.Amount)
				vals[ca] = vals[ca].Add(b.Value)
			}
		}
	}
	r.sections = append(r.sections, &Section{
		date:    d.Date,
		values:  vals,
		amounts: amts,
	})
}

func (r *Register) Sink(ctx context.Context, ch <-chan *ast.Day) error {
	for {
		d, ok, err := cpr.Pop(ctx, ch)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		r.Add(d)
	}
	return nil
}

// Render renders the register.
func (r *Register) Render(w io.Writer) error {
	var lenAcc, lenCom, lenAmt int
	for _, s := range r.sections {
		for ca, a := range s.amounts {
			if len(ca.Account.String()) > lenAcc {
				lenAcc = len(ca.Account.String())
			}
			if len(ca.Commodity.String()) > lenCom {
				lenCom = len(ca.Commodity.String())
			}
			if len(a.StringFixed(2)) > lenAmt {
				lenAmt = len(a.StringFixed(2))
			}
		}
	}

	for _, s := range r.sections {
		var counter int
		for ca, a := range s.amounts {
			if counter == 0 {
				fmt.Fprintf(w, "%s ", s.date.Format("2006-01-02"))
			} else {
				io.WriteString(w, "           ")
			}
			counter++
			fmt.Fprintf(w, "%-*s %*s %-*s\n", lenAcc, ca.Account.String(), lenAmt, a.StringFixed(2), lenCom, ca.Commodity.String())
		}
	}
	return nil
}

// Section represents one day in the register report.
type Section struct {
	date            time.Time
	amounts, values amounts.Amounts
}
