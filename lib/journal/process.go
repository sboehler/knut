package journal

import (
	"fmt"
	"strings"

	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/predicate"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal/printer"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/price"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/shopspring/decimal"
)

type DayFn = func(*Day) error

func NoOp[T any](_ T) error {
	return nil
}

// Error is a processing error, with a reference to a directive with
// a source location.
type Error struct {
	directive model.Directive
	msg       string
}

func (be Error) Error() string {
	var s strings.Builder
	s.WriteString(be.msg)
	s.WriteRune('\n')
	s.WriteRune('\n')
	p := printer.New(&s)
	p.PrintDirectiveLn(be.directive)
	return s.String()
}

// ComputePrices updates prices.
func ComputePrices(v *model.Commodity) DayFn {
	if v == nil {
		return NoOp[*Day]
	}
	var previous price.NormalizedPrices
	prc := make(price.Prices)
	proc := Processor{
		Price: func(p *model.Price) error {
			prc.Insert(p.Commodity, p.Price, p.Target)
			return nil
		},
		DayEnd: func(d *Day) error {
			if len(d.Prices) == 0 {
				d.Normalized = previous
			} else {
				d.Normalized = prc.Normalize(v)
				previous = d.Normalized
			}
			return nil
		},
	}
	return proc.Process
}

// Balance balances the journal.
func Check(reg *model.Registry, valuation *model.Commodity) DayFn {
	quantities := make(amounts.Amounts)
	accounts := set.New[*model.Account]()

	checker := Processor{

		Open: func(o *model.Open) error {
			if accounts.Has(o.Account) {
				return Error{o, "account is already open"}
			}
			accounts.Add(o.Account)
			return nil
		},

		Posting: func(t *model.Transaction, p *model.Posting) error {
			if !accounts.Has(p.Account) {
				return Error{t, fmt.Sprintf("account %s is not open", p.Account)}
			}
			if p.Account.IsAL() {
				quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
			}
			return nil
		},

		Assertion: func(a *model.Assertion) error {
			if !accounts.Has(a.Account) {
				return Error{a, "account is not open"}
			}
			position := amounts.AccountCommodityKey(a.Account, a.Commodity)
			if qty, ok := quantities[position]; !ok || !qty.Equal(a.Quantity) {
				return Error{a, fmt.Sprintf("failed assertion: account has position: %s %s", qty, position.Commodity.Name())}
			}
			return nil
		},

		Close: func(c *model.Close) error {
			for pos, amount := range quantities {
				if pos.Account != c.Account {
					continue
				}
				if !amount.IsZero() {
					return Error{c, fmt.Sprintf("account has nonzero position: %s %s", amount, pos.Commodity.Name())}
				}
				delete(quantities, pos)
			}
			if !accounts.Has(c.Account) {
				return Error{c, "account is not open"}
			}
			accounts.Remove(c.Account)
			return nil
		},
	}
	return checker.Process
}

// Balance balances the journal.
func Valuate(reg *model.Registry, valuation *model.Commodity) DayFn {

	var prevPrices, prices price.NormalizedPrices
	quantities := make(amounts.Amounts)

	valuator := Processor{

		DayStart: func(d *Day) error {
			prices = d.Normalized

			for pos, qty := range quantities {
				if pos.Commodity == valuation {
					continue
				}
				if !pos.Account.IsAL() {
					continue
				}
				if qty.IsZero() {
					continue
				}
				prevPrice, err := prevPrices.Price(pos.Commodity)
				if err != nil {
					return err
				}
				currentPrice, err := prices.Price(pos.Commodity)
				if err != nil {
					return err
				}
				delta := currentPrice.Sub(prevPrice)
				if delta.IsZero() {
					continue
				}
				gain := price.Multiply(delta, qty)
				credit := reg.Accounts().ValuationAccountFor(pos.Account)
				d.Transactions = append(d.Transactions, transaction.Builder{
					Date:        d.Date,
					Description: fmt.Sprintf("Adjust value of %s in account %s", pos.Commodity.Name(), pos.Account.Name()),
					Postings: posting.Builder{
						Credit:    credit,
						Debit:     pos.Account,
						Commodity: pos.Commodity,
						Value:     gain,
					}.Build(),
					Targets: []*model.Commodity{pos.Commodity},
				}.Build())
			}
			return nil
		},

		Posting: func(_ *model.Transaction, p *model.Posting) error {
			if p.Quantity.IsZero() {
				return nil
			}
			if p.Account.IsAL() {
				quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
			}
			v := p.Quantity
			if valuation != p.Commodity {
				var err error
				v, err = prices.Valuate(p.Commodity, p.Quantity)
				if err != nil {
					return err
				}
			}
			p.Value = v
			return nil
		},

		DayEnd: func(d *Day) error {
			prevPrices = d.Normalized
			return nil
		},
	}
	return valuator.Process
}

func Filter(part date.Partition) DayFn {
	proc := Processor{
		DayEnd: func(d *Day) error {
			if !part.Contains(d.Date) {
				d.Transactions = nil
			}
			return nil
		},
	}
	return proc.Process
}

// Balance balances the journal.
func CloseAccounts(j *Journal, enable bool, partition date.Partition) DayFn {
	if !enable {
		return func(d *Day) error { return nil }
	}

	quantities, values := make(amounts.Amounts), make(amounts.Amounts)
	closingDays := set.New[*Day]()
	for _, d := range partition.StartDates() {
		// j.Day creates the entry for the given date as a side effect.
		closingDays.Add(j.Day(d))
	}
	equityAccount := j.Registry.Accounts().MustGet("Equity:Equity")

	closer := Processor{
		DayStart: func(d *Day) error {
			if !closingDays.Has(d) {
				return nil
			}
			for k, quantity := range quantities {
				if quantity.IsZero() && values[k].IsZero() {
					continue
				}
				d.Transactions = append(d.Transactions, transaction.Builder{
					Date:        d.Date,
					Description: fmt.Sprintf("Closing account %s in %s", k.Account.Name(), k.Commodity.Name()),
					Postings: posting.Builder{
						Credit:    k.Account,
						Debit:     equityAccount,
						Commodity: k.Commodity,
						Quantity:  quantity,
						Value:     values[k],
					}.Build(),
				}.Build())
			}
			return nil
		},
		Posting: func(_ *model.Transaction, p *model.Posting) error {
			if p.Account.IsAL() {
				return nil
			}
			if p.Account == equityAccount {
				return nil
			}
			quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
			values.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Value)
			return nil
		},
	}
	return closer.Process
}

// Sort sorts the directives in this day.
func Sort() DayFn {
	proc := Processor{
		DayEnd: func(d *Day) error {
			compare.Sort(d.Transactions, transaction.Compare)
			return nil
		},
	}
	return proc.Process
}

type Collection interface {
	Insert(k amounts.Key, v decimal.Decimal)
}

type Query struct {
	Mapper    mapper.Mapper[amounts.Key]
	Predicate predicate.Predicate[amounts.Key]
	Valuation *model.Commodity
}

func (query Query) Execute(c Collection) DayFn {
	if query.Predicate == nil {
		query.Predicate = predicate.True[amounts.Key]
	}
	if query.Mapper == nil {
		query.Mapper = mapper.Identity[amounts.Key]
	}
	querier := Processor{
		Posting: func(t *model.Transaction, b *model.Posting) error {
			amount := b.Quantity
			if query.Valuation != nil {
				amount = b.Value
			}
			kc := amounts.Key{
				Date:        t.Date,
				Account:     b.Account,
				Other:       b.Other,
				Commodity:   b.Commodity,
				Valuation:   query.Valuation,
				Description: t.Description,
			}
			if query.Predicate(kc) {
				c.Insert(query.Mapper(kc), amount)
			}
			return nil
		},
	}
	return querier.Process
}
