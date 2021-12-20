package process

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/ast/printer"
	"github.com/sboehler/knut/lib/journal/past"
)

// Processor processes ASTs.
type Processor struct {

	// The context of this journal.
	Context journal.Context

	// Filter applies the given filter to postings of transactions
	// and assertions.
	Filter journal.Filter

	// Expand controls whether Accrual add-ons are expanded.
	Expand bool
}

// Process processes an AST to a PAST. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr Processor) Process(a *ast.AST) (*past.PAST, error) {
	var astCp = &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for d, day := range a.Days {
		dayCp := astCp.Day(d)

		dayCp.Openings = day.Openings

		dayCp.Prices = day.Prices

		for _, trx := range day.Transactions {
			pr.processTransaction(astCp, trx)
		}

		for _, a := range day.Assertions {
			pr.processAssertion(astCp, a)
		}

		dayCp.Closings = day.Closings
	}

	var (
		sorted  = astCp.SortedDays()
		amounts past.Amounts
		acc     = make(past.Accounts)
		res     = &past.PAST{
			Context: a.Context,
		}
	)
	for _, d := range sorted {
		day := &past.Day{
			Date:       d.Date,
			AST:        a.Days[d.Date], // possibly nil
			Amounts:    amounts.Clone(),
			Openings:   d.Openings,
			Prices:     d.Prices,
			Assertions: d.Assertions,
			Values:     d.Values,
			Closings:   d.Closings,
		}
		res.Days = append(res.Days, day)
		for _, o := range d.Openings {
			if err := acc.Open(o.Account); err != nil {
				return nil, err
			}
		}
		for _, t := range d.Transactions {
			for _, p := range t.Postings {
				if !acc.IsOpen(p.Credit) {
					return nil, Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
				}
				if !acc.IsOpen(p.Debit) {
					return nil, Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
				}
				day.Amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
			}
			day.Transactions = append(day.Transactions, t)
		}
		if dayA, ok := a.Days[d.Date]; ok {
			for _, v := range dayA.Values {
				if !acc.IsOpen(v.Account) {
					return nil, Error{v, "account is not open"}
				}
				var (
					t   *ast.Transaction
					err error
				)
				if t, err = pr.processValue(day.Amounts, v); err != nil {
					return nil, err
				}
				for _, p := range t.Postings {
					if !acc.IsOpen(p.Credit) {
						return nil, Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
					}
					if !acc.IsOpen(p.Debit) {
						return nil, Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
					}
					day.Amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
				}
				day.Transactions = append(day.Transactions, t)
			}
		}
		for _, a := range d.Assertions {
			if !acc.IsOpen(a.Account) {
				return nil, Error{a, "account is not open"}
			}
			var pos = past.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
			va, ok := day.Amounts[pos]
			if !ok || !va.Equal(a.Amount) {
				return nil, Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, pos.Commodity)}
			}
		}
		for _, c := range d.Closings {
			for pos, amount := range day.Amounts {
				if pos.Account != c.Account {
					continue
				}
				if !amount.IsZero() {
					return nil, Error{c, "account has nonzero position"}
				}
				delete(amounts, pos)
			}
			if err := acc.Close(c.Account); err != nil {
				return nil, err
			}
		}
		amounts = day.Amounts
	}
	return res, nil
}

// Process2 processes an AST to a PAST. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr Processor) Process2(a *ast.AST) (*past.PAST, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dayCh, errCh := pr.ProcessAsync(ctx, a)
	res := &past.PAST{
		Context: a.Context,
	}
	for {
		select {
		case day, ok := <-dayCh:
			if !ok {
				dayCh = nil
				break
			}
			res.Days = append(res.Days, day)

		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				break
			}
			return nil, err
		}
		if dayCh == nil && errCh == nil {
			break
		}
	}
	return res, nil

}

// ProcessAsync processes an AST to a stream of past.Day. It check assertions
// and the usage of open and closed accounts. It will also
// resolve Value directives and convert them to transactions.
func (pr Processor) ProcessAsync(ctx context.Context, a *ast.AST) (<-chan *past.Day, <-chan error) {
	var astCp = &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for d, day := range a.Days {
		dayCp := astCp.Day(d)

		dayCp.Openings = day.Openings

		dayCp.Prices = day.Prices

		for _, trx := range day.Transactions {
			pr.processTransaction(astCp, trx)
		}

		for _, a := range day.Assertions {
			pr.processAssertion(astCp, a)
		}

		dayCp.Closings = day.Closings
	}
	var (
		errCh = make(chan error)
		resCh = make(chan *past.Day)

		errOrExit = func(err error) bool {
			select {
			case errCh <- err:
				return false
			case <-ctx.Done():
				return true
			}
		}
	)

	go func() {
		defer close(resCh)
		defer close(errCh)
		var (
			sorted  = astCp.SortedDays()
			amounts past.Amounts
			acc     = make(past.Accounts)
			res     = &past.PAST{
				Context: a.Context,
			}
		)
		for _, d := range sorted {
			day := &past.Day{
				Date:       d.Date,
				AST:        a.Days[d.Date], // possibly nil
				Amounts:    amounts.Clone(),
				Openings:   d.Openings,
				Prices:     d.Prices,
				Assertions: d.Assertions,
				Values:     d.Values,
				Closings:   d.Closings,
			}
			res.Days = append(res.Days, day)
			for _, o := range d.Openings {
				if err := acc.Open(o.Account); err != nil && errOrExit(err) {
					return
				}
			}
			for _, t := range d.Transactions {
				for _, p := range t.Postings {
					if !acc.IsOpen(p.Credit) {
						if errOrExit(Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}) {
							return
						}
					}
					if !acc.IsOpen(p.Debit) {
						if errOrExit(Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}) {
							return
						}
					}
					day.Amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
				}
				day.Transactions = append(day.Transactions, t)
			}
			if dayA, ok := a.Days[d.Date]; ok {
				for _, v := range dayA.Values {
					if !acc.IsOpen(v.Account) {
						if errOrExit(Error{v, "account is not open"}) {
							return
						}
					}
					var (
						t   *ast.Transaction
						err error
					)
					if t, err = pr.processValue(day.Amounts, v); err != nil {
						errOrExit(err)
						return
					}
					for _, p := range t.Postings {
						if !acc.IsOpen(p.Credit) {
							if errOrExit(Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}) {
								return
							}
						}
						if !acc.IsOpen(p.Debit) {
							if errOrExit(Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}) {
								return
							}
						}
						day.Amounts.Book(p.Credit, p.Debit, p.Amount, p.Commodity)
					}
					day.Transactions = append(day.Transactions, t)
				}
			}
			for _, a := range d.Assertions {
				if !acc.IsOpen(a.Account) {
					errCh <- Error{a, "account is not open"}
				}
				var pos = past.CommodityAccount{Account: a.Account, Commodity: a.Commodity}
				va, ok := day.Amounts[pos]
				if !ok || !va.Equal(a.Amount) {
					if errOrExit(Error{a, fmt.Sprintf("assertion failed: account %s has %s %s", a.Account, va, pos.Commodity)}) {
						return
					}
				}
			}
			for _, c := range d.Closings {
				for pos, amount := range day.Amounts {
					if pos.Account != c.Account {
						continue
					}
					if !amount.IsZero() && errOrExit(Error{c, "account has nonzero position"}) {
						return
					}
					delete(amounts, pos)
				}
				if err := acc.Close(c.Account); err != nil && errOrExit(err) {
					return
				}
			}
			amounts = day.Amounts
			select {
			case resCh <- day:
			case <-ctx.Done():
				return
			}
		}
	}()
	return resCh, errCh
}

// ProcessTransaction adds a transaction directive.
func (pr *Processor) processTransaction(a *ast.AST, t *ast.Transaction) {
	if pr.Expand && len(t.AddOns) > 0 {
		for _, addOn := range t.AddOns {
			switch acc := addOn.(type) {
			case *ast.Accrual:
				for _, ts := range acc.Expand(t) {
					pr.processTransaction(a, ts)
				}
			}
		}
	} else {
		var filtered []ast.Posting
		for _, p := range t.Postings {
			if p.Matches(pr.Filter) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == len(t.Postings) {
			a.AddTransaction(t)
		} else if len(filtered) > 0 && len(filtered) < len(t.Postings) {
			a.AddTransaction(&ast.Transaction{
				Range:       t.Range,
				Date:        t.Date,
				Description: t.Description,
				Postings:    filtered,
				Tags:        t.Tags,
			})
		}
	}
}

// ProcessAssertion adds an assertion directive.
func (pr *Processor) processAssertion(as *ast.AST, a *ast.Assertion) {
	if pr.Filter.MatchAccount(a.Account) && pr.Filter.MatchCommodity(a.Commodity) {
		as.AddAssertion(a)
	}
}

func (pr *Processor) processValue(bal past.Amounts, v *ast.Value) (*ast.Transaction, error) {
	valAcc, err := pr.Context.ValuationAccountFor(v.Account)
	if err != nil {
		return nil, err
	}
	return &ast.Transaction{
		Date:        v.Date,
		Description: fmt.Sprintf("Valuation adjustment for %v in %v", v.Commodity, v.Account),
		Tags:        nil,
		Postings: []ast.Posting{
			ast.NewPosting(valAcc, v.Account, v.Commodity, v.Amount.Sub(bal.Amount(v.Account, v.Commodity))),
		},
	}, nil
}

// ASTFromPath reads directives from the given channel and
// builds a Ledger if successful.
func (pr *Processor) ASTFromPath(p string) (*ast.AST, error) {
	par := parser.RecursiveParser{
		File:    p,
		Context: pr.Context,
	}
	results := par.Parse()
	var b = &ast.AST{
		Context: pr.Context,
		Days:    make(map[time.Time]*ast.Day),
	}
	for res := range results {
		switch t := res.(type) {
		case error:
			return nil, t
		case *ast.Open:
			b.AddOpen(t)
		case *ast.Price:
			b.AddPrice(t)
		case *ast.Transaction:
			b.AddTransaction(t)
		case *ast.Assertion:
			b.AddAssertion(t)
		case *ast.Value:
			b.AddValue(t)
		case *ast.Close:
			b.AddClose(t)
		default:
			return nil, fmt.Errorf("unknown: %#v", t)
		}
	}
	return b, nil
}

// PASTFromPath processes a journal and returns a processed AST.
func (pr *Processor) PASTFromPath(p string) (*past.PAST, error) {
	as, err := pr.ASTFromPath(p)
	if err != nil {
		return nil, err
	}
	return pr.Process2(as)
}

// Error is an error.
type Error struct {
	directive ast.Directive
	msg       string
}

func (be Error) Error() string {
	var (
		p printer.Printer
		b strings.Builder
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	p.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}
