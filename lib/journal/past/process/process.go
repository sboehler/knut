package process

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sboehler/knut/lib/balance/prices"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/ast/parser"
	"github.com/sboehler/knut/lib/journal/ast/printer"
	"github.com/sboehler/knut/lib/journal/past"
	"github.com/sboehler/knut/lib/journal/val"
	"github.com/shopspring/decimal"
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

	// Valuation is the valuation commodity.
	Valuation *journal.Commodity
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
		dayCp.Closings = day.Closings

		for _, trx := range day.Transactions {
			pr.processTransaction(astCp, trx)
		}

		for _, a := range day.Assertions {
			pr.processAssertion(astCp, a)
		}

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
			select {
			case resCh <- day:
				amounts = day.Amounts
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

// Valuate computes prices.
func (pr Processor) Valuate(ctx context.Context, inCh chan *past.Day) chan *val.Day {
	var (
		resCh = make(chan *val.Day)
		prc   = make(prices.Prices)
	)
	go func() {
		defer close(resCh)

		var previous *val.Day
		for {
			select {

			case day, ok := <-inCh:
				if !ok {
					return
				}
				vday := &val.Day{
					Date: day.Date,
					Day:  day,
				}
				if pr.Valuation == nil {
					break
				}
				if day.AST != nil && len(day.AST.Prices) > 0 {
					for _, p := range day.AST.Prices {
						prc.Insert(p)
					}
					vday.Prices = prc.Normalize(pr.Valuation)
				} else if previous != nil {
					vday.Prices = previous.Prices
				}

				select {
				case resCh <- vday:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return resCh
}

// ValuateTransactions computes prices.
func (pr Processor) ValuateTransactions(ctx context.Context, inCh chan *val.Day) (chan *val.Day, chan error) {

	var (
		errCh = make(chan error)
		resCh = make(chan *val.Day)

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

		var values = make(past.Amounts)

		for {
			select {

			case day, ok := <-inCh:
				if !ok {
					return
				}
				day.Values = values

				for _, t := range day.Day.Transactions {
					tv, err := pr.valuateAndBookTransaction(day, t)
					if err != nil {
						errOrExit(err)
						return
					}
					day.Transactions = append(day.Transactions, tv)
				}

				pr.computeValuationTransactions(day)

				values = values.Clone()
				select {
				case resCh <- day:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return

			}
		}

	}()

	return resCh, errCh
}

func (pr Processor) valuateAndBookTransaction(b *val.Day, t *ast.Transaction) (*val.Transaction, error) {
	var postings []val.Posting
	for i, posting := range t.Postings {
		var (
			value     decimal.Decimal
			commodity *journal.Commodity
			err       error
		)
		if pr.Valuation == nil || pr.Valuation == posting.Commodity {
			commodity = posting.Commodity
			value = posting.Amount
		} else {
			if value, err = b.Prices.Valuate(posting.Commodity, posting.Amount); err != nil {
				return nil, Error{t, fmt.Sprintf("no price found for commodity %s", posting.Commodity)}
			}
		}
		b.Values.Book(posting.Credit, posting.Debit, value, commodity)
		postings = append(postings, val.Posting{
			Source:    &t.Postings[i],
			Credit:    posting.Credit,
			Debit:     posting.Debit,
			Value:     value,
			Commodity: commodity,
		})
	}
	return &val.Transaction{
		Source:   t,
		Postings: postings,
	}, nil
}

// computeValuationTransactions checks whether the valuation for the positions
// corresponds to the amounts. If not, the difference is due to a valuation
// change of the previous amount, and a transaction is created to adjust the
// valuation.
func (pr Processor) computeValuationTransactions(b *val.Day) {
	if pr.Valuation == nil {
		return
	}
	for pos, va := range b.Day.Amounts {
		if pos.Commodity == pr.Valuation {
			continue
		}
		var at = pos.Account.Type()
		if at != journal.ASSETS && at != journal.LIABILITIES {
			continue
		}
		value, err := b.Prices.Valuate(pos.Commodity, va)
		if err != nil {
			panic(fmt.Sprintf("no valuation found for commodity %s", pos.Commodity))
		}
		var diff = value.Sub(b.Values[pos])
		if diff.IsZero() {
			continue
		}
		valAcc, err := pr.Context.ValuationAccountFor(pos.Account)
		if err != nil {
			panic(fmt.Sprintf("could not obtain valuation account for account %s", pos.Account))
		}
		// create a transaction to adjust the valuation
		b.Transactions = append(b.Transactions, &val.Transaction{
			Source: nil,
			Postings: []val.Posting{
				{
					Value:     diff,
					Credit:    valAcc,
					Debit:     pos.Account,
					Commodity: pos.Commodity,
				},
			},
		})
	}
}
