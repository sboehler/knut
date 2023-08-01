package transaction

import (
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

// Transaction represents a transaction.
type Transaction struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*posting.Posting
	Targets     []*commodity.Commodity
}

// Less defines an order on transactions.
func Compare(t *Transaction, t2 *Transaction) compare.Order {
	if o := compare.Time(t.Date, t2.Date); o != compare.Equal {
		return o
	}
	if o := compare.Ordered(t.Description, t2.Description); o != compare.Equal {
		return o
	}
	for i := 0; i < len(t.Postings) && i < len(t2.Postings); i++ {
		if o := posting.Compare(t.Postings[i], t2.Postings[i]); o != compare.Equal {
			return o
		}
	}
	return compare.Ordered(len(t.Postings), len(t2.Postings))
}

// Builder builds transactions.
type Builder struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*posting.Posting
	Targets     []*commodity.Commodity
}

// Build builds a transactions.
func (tb Builder) Build() *Transaction {
	return &Transaction{
		Src:         tb.Src,
		Date:        tb.Date,
		Description: tb.Description,
		Postings:    tb.Postings,
		Targets:     tb.Targets,
	}
}

func Create(reg *registry.Registry, t *syntax.Transaction) ([]*Transaction, error) {
	date, err := parseDate(t.Date)
	if err != nil {
		return nil, err
	}
	desc := t.Description.Content.Extract()
	postings, err := posting.Create(reg, t.Bookings)
	if err != nil {
		return nil, err
	}
	var targets []*commodity.Commodity
	if !t.Addons.Performance.Empty() {
		targets = []*commodity.Commodity{}
		for _, c := range t.Addons.Performance.Targets {
			com, err := reg.Commodities().Create(c)
			if err != nil {
				return nil, err
			}
			targets = append(targets, com)
		}
	}
	res := Builder{
		Src:         t,
		Date:        date,
		Description: desc,
		Postings:    postings,
		Targets:     targets,
	}.Build()
	if !t.Addons.Accrual.Empty() {
		return expand(reg, res, &t.Addons.Accrual)
	}
	return []*Transaction{res}, nil

}

// Expand expands an accrual transaction.
func expand(reg *registry.Registry, t *Transaction, accrual *syntax.Accrual) ([]*Transaction, error) {
	account, err := reg.Accounts().Create(accrual.Account)
	if err != nil {
		return nil, err
	}
	start, err := parseDate(accrual.Start)
	if err != nil {
		return nil, err
	}
	end, err := parseDate(accrual.End)
	if err != nil {
		return nil, err
	}
	interval, err := date.ParseInterval(accrual.Interval.Extract())
	if err != nil {
		return nil, syntax.Error{
			Message: "parsing interval",
			Range:   accrual.Interval.Range,
			Wrapped: err,
		}
	}
	var result []*Transaction
	for _, p := range t.Postings {
		if p.Account.IsAL() {
			result = append(result, Builder{
				Src:         t.Src,
				Date:        t.Date,
				Description: t.Description,
				Postings: posting.Builder{
					Credit:    account,
					Debit:     p.Account,
					Commodity: p.Commodity,
					Amount:    p.Amount,
				}.Build(),
				Targets: t.Targets,
			}.Build())
		}
		if p.Account.IsIE() {
			partition := date.NewPartition(date.Period{Start: start, End: end}, interval, 0)
			amount, rem := p.Amount.QuoRem(decimal.NewFromInt(int64(partition.Size())), 1)
			for i, dt := range partition.EndDates() {
				a := amount
				if i == 0 {
					a = a.Add(rem)
				}
				result = append(result, Builder{
					Src:         t.Src,
					Date:        dt,
					Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, partition.Size()),
					Postings: posting.Builder{
						Credit:    account,
						Debit:     p.Account,
						Commodity: p.Commodity,
						Amount:    a,
					}.Build(),
					Targets: t.Targets,
				}.Build())
			}
		}
	}
	return result, nil
}

func parseDate(d syntax.Date) (time.Time, error) {
	date, err := time.Parse("2006-01-02", d.Extract())
	if err != nil {
		return date, syntax.Error{
			Message: "parsing date",
			Range:   d.Range,
			Wrapped: err,
		}
	}
	return date, nil
}
