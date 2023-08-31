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
	return func(day *Day) error {
		if len(day.Prices) == 0 {
			day.Normalized = previous
		} else {
			for _, p := range day.Prices {
				prc.Insert(p.Commodity, p.Price, p.Target)
			}
			day.Normalized = prc.Normalize(v)
			previous = day.Normalized
		}
		return nil
	}
}

// Balance balances the journal.
func Balance(reg *model.Registry, v *model.Commodity) DayFn {
	quantities, values := make(amounts.Amounts), make(amounts.Amounts)
	accounts := set.New[*model.Account]()

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
				if !accounts.Has(p.Account) {
					return Error{t, fmt.Sprintf("account %s is not open", p.Account)}
				}
				if p.Account.IsAL() {
					quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
				}
			}
		}
		return nil
	}

	processAssertions := func(d *Day) error {
		for _, a := range d.Assertions {
			if !accounts.Has(a.Account) {
				return Error{a, "account is not open"}
			}
			position := amounts.AccountCommodityKey(a.Account, a.Commodity)
			if va, ok := quantities[position]; !ok || !va.Equal(a.Quantity) {
				return Error{a, fmt.Sprintf("failed assertion: account has position: %s %s", va, position.Commodity.Name())}
			}
		}
		return nil
	}

	processClosings := func(d *Day) error {
		for _, c := range d.Closings {
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
		}
		return nil
	}

	valuateTransactions := func(d *Day) error {
		for _, t := range d.Transactions {
			for _, posting := range t.Postings {
				if v != posting.Commodity {
					v, err := d.Normalized.Valuate(posting.Commodity, posting.Quantity)
					if err != nil {
						return err
					}
					posting.Value = v
				} else {
					posting.Value = posting.Quantity
				}
				if posting.Account.IsAL() {
					values.Add(amounts.AccountCommodityKey(posting.Account, posting.Commodity), posting.Value)
				}
			}
		}
		return nil
	}

	valuateGains := func(d *Day) error {
		for pos, qty := range quantities {
			if pos.Commodity == v {
				continue
			}
			if !pos.Account.IsAL() {
				continue
			}
			value, err := d.Normalized.Valuate(pos.Commodity, qty)
			if err != nil {
				return fmt.Errorf("no valuation found for commodity %s", pos.Commodity.Name())
			}
			gain := value.Sub(values[pos])
			if gain.IsZero() {
				continue
			}
			credit := reg.ValuationAccountFor(pos.Account)
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
			values.Add(pos, gain)
			values.Add(amounts.AccountCommodityKey(credit, pos.Commodity), gain.Neg())
		}
		return nil

	}

	return func(d *Day) error {
		if err := processOpenings(d); err != nil {
			return err
		}
		if err := processTransactions(d); err != nil {
			return err
		}
		if err := processAssertions(d); err != nil {
			return err
		}
		if v != nil {
			if err := valuateTransactions(d); err != nil {
				return err
			}
			if err := valuateGains(d); err != nil {
				return err
			}
		}
		if err := processClosings(d); err != nil {
			return err
		}
		return nil
	}
}

func Filter(part date.Partition) DayFn {
	return func(d *Day) error {
		if !part.Contains(d.Date) {
			d.Transactions = nil
		}
		return nil
	}
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
	return func(d *Day) error {
		if closingDays.Has(d) {
			for k, quantity := range quantities {
				if k.Account.IsAL() {
					continue
				}
				if k.Account == equityAccount {
					continue
				}
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
		}
		for _, t := range d.Transactions {
			for _, p := range t.Postings {
				if p.Account.IsAL() {
					continue
				}
				if p.Account == equityAccount {
					continue
				}
				quantities.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Quantity)
				values.Add(amounts.AccountCommodityKey(p.Account, p.Commodity), p.Value)
			}
		}
		return nil
	}
}

// Sort sorts the directives in this day.
func Sort() DayFn {
	return func(d *Day) error {
		compare.Sort(d.Transactions, transaction.Compare)
		return nil
	}
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
	return func(d *Day) error {
		for _, t := range d.Transactions {
			for _, b := range t.Postings {
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
			}
		}
		return nil
	}
}
