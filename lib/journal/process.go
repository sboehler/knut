package journal

import (
	"fmt"
	"strings"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/shopspring/decimal"
)

type DayFn func(*Day, func(*Day)) error

func NoOp[T any](d T, next func(T)) error {
	next(d)
	return nil
}

// Error is a processing error, with a reference to a directive with
// a source location.
type Error struct {
	directive Directive
	msg       string
}

func (be Error) Error() string {
	var (
		p Printer
		b strings.Builder
	)
	fmt.Fprintf(&b, "%s:\n", be.directive.Position().Start)
	p.PrintDirective(&b, be.directive)
	fmt.Fprintf(&b, "\n%s\n", be.msg)
	return b.String()
}

// ComputePrices updates prices.
func ComputePrices(v *Commodity) DayFn {
	if v == nil {
		return NoOp[*Day]
	}
	var previous NormalizedPrices
	prc := make(Prices)
	return func(day *Day, next func(*Day)) error {
		if len(day.Prices) == 0 {
			day.Normalized = previous
		} else {
			for _, p := range day.Prices {
				prc.Insert(p.Commodity, p.Price, p.Target)
			}
			day.Normalized = prc.Normalize(v)
			previous = day.Normalized
		}
		next(day)
		return nil
	}
}

// Balance balances the
func Balance(jctx Context) DayFn {
	amounts := make(Amounts)
	accounts := set.New[*Account]()

	processOpenings := func(d *Day) error {
		for _, o := range d.Openings {
			if accounts.Has(o.Account) {
				return Error{o, "account is already open"}
			}
			accounts.Add(o.Account)
		}
		return nil
	}

	processTransactions := func(d *Day) error {
		for _, t := range d.Transactions {
			for _, p := range t.Postings {
				if !accounts.Has(p.Credit) {
					return Error{t, fmt.Sprintf("credit account %s is not open", p.Credit)}
				}
				if !accounts.Has(p.Debit) {
					return Error{t, fmt.Sprintf("debit account %s is not open", p.Debit)}
				}
				amounts.Add(AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
				amounts.Add(AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
			}
		}
		return nil
	}

	processValues := func(d *Day) error {
		for _, v := range d.Values {
			if !accounts.Has(v.Account) {
				return Error{v, "account is not open"}
			}
			valAcc := jctx.ValuationAccountFor(v.Account)
			p := PostingWithTargets(valAcc, v.Account, v.Commodity, v.Amount.Sub(amounts.Amount(AccountCommodityKey(v.Account, v.Commodity))), []*Commodity{v.Commodity})
			d.Transactions = append(d.Transactions, TransactionBuilder{
				Date:        v.Date,
				Description: fmt.Sprintf("Valuation adjustment for %s in %s", v.Commodity.Name(), v.Account.Name()),
				Postings:    []*Posting{p},
			}.Build())
			amounts.Add(AccountCommodityKey(p.Credit, p.Commodity), p.Amount.Neg())
			amounts.Add(AccountCommodityKey(p.Debit, p.Commodity), p.Amount)
		}
		compare.Sort(d.Transactions, CompareTransactions)
		return nil
	}

	processAssertions := func(d *Day) error {
		for _, a := range d.Assertions {
			if !accounts.Has(a.Account) {
				return Error{a, "account is not open"}
			}
			position := AccountCommodityKey(a.Account, a.Commodity)
			if va, ok := amounts[position]; !ok || !va.Equal(a.Amount) {
				return Error{a, fmt.Sprintf("account has position: %s %s", va, position.Commodity.Name())}
			}
		}
		return nil
	}

	processClosings := func(d *Day) error {
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
			if !accounts.Has(c.Account) {
				return Error{c, "account is not open"}
			}
			accounts.Remove(c.Account)
		}
		return nil
	}

	return func(d *Day, next func(*Day)) error {
		if err := processOpenings(d); err != nil {
			return err
		}
		if err := processTransactions(d); err != nil {
			return err
		}
		if err := processValues(d); err != nil {
			return err
		}
		if err := processAssertions(d); err != nil {
			return err
		}
		if err := processClosings(d); err != nil {
			return err
		}
		d.Amounts = amounts.Clone()
		next(d)
		return nil
	}
}

// Valuate valuates the
func Valuate(jctx Context, v *Commodity) DayFn {
	if v == nil {
		return NoOp[*Day]
	}
	values := make(Amounts)

	valuateTransactions := func(d *Day) error {
		for _, t := range d.Transactions {
			for _, posting := range t.Postings {
				if v != posting.Commodity {
					v, err := d.Normalized.Valuate(posting.Commodity, posting.Amount)
					if err != nil {
						return err
					}
					posting.Value = v
				} else {
					posting.Value = posting.Amount
				}
				values.Add(AccountCommodityKey(posting.Credit, posting.Commodity), posting.Value.Neg())
				values.Add(AccountCommodityKey(posting.Debit, posting.Commodity), posting.Value)
			}
		}
		return nil
	}

	valuateGains := func(d *Day) error {
		for pos, amt := range d.Amounts {
			if pos.Commodity == v {
				continue
			}
			if !pos.Account.IsAL() {
				continue
			}
			value, err := d.Normalized.Valuate(pos.Commodity, amt)
			if err != nil {
				return fmt.Errorf("no valuation found for commodity %s", pos.Commodity.Name())
			}
			gain := value.Sub(values[pos])
			if gain.IsZero() {
				continue
			}
			credit := jctx.ValuationAccountFor(pos.Account)
			d.Transactions = append(d.Transactions, TransactionBuilder{
				Date:        d.Date,
				Description: fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity.Name(), pos.Account.Name()),
				Postings: []*Posting{
					NewValuePosting(credit, pos.Account, pos.Commodity, gain, []*Commodity{pos.Commodity}),
				},
			}.Build())
			values.Add(pos, gain)
			values.Add(AccountCommodityKey(credit, pos.Commodity), gain.Neg())
		}
		return nil

	}

	return func(d *Day, next func(*Day)) error {
		if err := valuateTransactions(d); err != nil {
			return err
		}
		if err := valuateGains(d); err != nil {
			return err
		}
		d.Value = values.Clone()
		next(d)
		return nil
	}
}

// Sort sorts the directives in this day.
func Sort() DayFn {
	return func(d *Day, next func(*Day)) error {
		compare.Sort(d.Transactions, CompareTransactions)
		next(d)
		return nil
	}
}

type Collection interface {
	Insert(k Key, v decimal.Decimal)
}

func Aggregate(m mapper.Mapper[Key], f filter.Filter[Key], v *Commodity, c Collection) DayFn {
	if f == nil {
		f = filter.AllowAll[Key]
	}
	if m == nil {
		m = mapper.Identity[Key]
	}
	return func(d *Day, next func(*Day)) error {
		for _, t := range d.Transactions {
			for _, b := range t.Postings {
				amt := b.Amount
				if v != nil {
					amt = b.Value
				}
				kc := Key{
					Date:        t.Date,
					Account:     b.Credit,
					Other:       b.Debit,
					Commodity:   b.Commodity,
					Valuation:   v,
					Description: t.Description,
				}
				if f(kc) {
					c.Insert(m(kc), amt.Neg())
				}
				kd := Key{
					Date:        t.Date,
					Account:     b.Debit,
					Other:       b.Credit,
					Commodity:   b.Commodity,
					Valuation:   v,
					Description: t.Description,
				}
				if f(kd) {
					c.Insert(m(kd), amt)
				}
			}
		}
		next(d)
		return nil
	}
}
