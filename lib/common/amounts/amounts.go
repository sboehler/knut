package amounts

import (
	"regexp"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Date           time.Time
	Account, Other *journal.Account
	Commodity      *journal.Commodity
	Valuation      *journal.Commodity
}

func DateKey(d time.Time) Key {
	return Key{Date: d}
}

func CommodityKey(c *journal.Commodity) Key {
	return Key{Commodity: c}
}

func AccountKey(a *journal.Account) Key {
	return Key{Account: a}
}

func AccountCommodityKey(a *journal.Account, c *journal.Commodity) Key {
	return Key{Account: a, Commodity: c}
}

// Amounts keeps track of amounts by account and commodity.
type Amounts map[Key]decimal.Decimal

// Amount returns the amount for the given key.
func (am Amounts) Amount(k Key) decimal.Decimal {
	return am[k]
}

func (am Amounts) Add(k Key, d decimal.Decimal) {
	am[k] = am[k].Add(d)
}

// Clone clones these amounts.
func (am Amounts) Clone() Amounts {
	clone := make(Amounts)
	for ca, v := range am {
		clone[ca] = v
	}
	return clone
}

// Minus mutably subtracts.
func (am Amounts) Minus(a Amounts) Amounts {
	for ca, v := range a {
		am[ca] = am[ca].Sub(v)
	}
	return am
}

func (am Amounts) Index(cmp compare.Compare[Key]) []Key {
	res := make([]Key, 0, len(am))
	for k := range am {
		res = append(res, k)
	}
	if cmp != nil {
		sort.Slice(res, func(i, j int) bool {
			return cmp(res[i], res[j]) == compare.Smaller
		})
	}
	return res
}

func (am Amounts) Commodities() map[*journal.Commodity]struct{} {
	cs := make(map[*journal.Commodity]struct{})
	for k := range am {
		cs[k.Commodity] = struct{}{}
	}
	return cs
}

func (am Amounts) CommoditiesSorted() []*journal.Commodity {
	cs := am.Commodities()
	return dict.SortedKeys(cs, journal.CompareCommodities)
}

func (am Amounts) Dates() map[time.Time]struct{} {
	cs := make(map[time.Time]struct{})
	for k := range am {
		cs[k.Date] = struct{}{}
	}
	return cs
}

func (am Amounts) DatesSorted() []time.Time {
	cs := am.Dates()
	return dict.SortedKeys(cs, compare.Time)
}

func (am Amounts) SumBy(f func(k Key) bool, m func(k Key) Key) Amounts {
	res := make(Amounts)
	if f == nil {
		f = filter.Default[Key]
	}
	if m == nil {
		m = Identity[Key]
	}
	for k, v := range am {
		if !f(k) {
			continue
		}
		kn := m(k)
		res[kn] = res[kn].Add(v)
	}
	return res
}

func (am Amounts) SumOver(f func(k Key) bool) decimal.Decimal {
	var res decimal.Decimal
	for k, v := range am {
		if !f(k) {
			continue
		}
		res = res.Add(v)
	}
	return res
}

type Mapper func(Key) Key

type KeyMapper struct {
	Date                 func(time.Time) time.Time
	Account, Other       func(*journal.Account) *journal.Account
	Commodity, Valuation func(*journal.Commodity) *journal.Commodity
}

func (km KeyMapper) Build() Mapper {
	return func(k Key) Key {
		var res Key
		if km.Date != nil {
			res.Date = km.Date(k.Date)
		}
		if km.Account != nil {
			res.Account = km.Account(k.Account)
		}
		if km.Other != nil {
			res.Other = km.Other(k.Other)
		}
		if km.Commodity != nil {
			res.Commodity = km.Commodity(k.Commodity)
		}
		if km.Valuation != nil {
			res.Valuation = km.Valuation(k.Valuation)
		}
		return res
	}
}

func Identity[T any](t T) T {
	return t
}

func FilterDates(t time.Time) filter.Filter[Key] {
	return func(k Key) bool {
		return !k.Date.After(t)
	}
}

func FilterCommodity(r *regexp.Regexp) filter.Filter[Key] {
	if r == nil {
		return filter.Default[Key]
	}
	return func(k Key) bool {
		return r.MatchString(k.Commodity.String())
	}
}

func FilterAccount(r *regexp.Regexp) filter.Filter[Key] {
	if r == nil {
		return filter.Default[Key]
	}
	return func(k Key) bool {
		return r.MatchString(k.Account.String())
	}
}

func FilterOther(r *regexp.Regexp) filter.Filter[Key] {
	if r == nil {
		return filter.Default[Key]
	}
	return func(k Key) bool {
		return r.MatchString(k.Other.String())
	}
}

func SortByDate(k1, k2 Key) compare.Order {
	return compare.Time(k1.Date, k2.Date)
}

func SortByAccount(jctx journal.Context, w map[*journal.Account]float64) compare.Compare[Key] {
	s := journal.CompareWeighted(jctx, w)
	return func(k1, k2 Key) compare.Order {
		return s(k1.Account, k2.Account)
	}
}

func SortByCommodity(k1, k2 Key) compare.Order {
	return compare.Ordered(k1.Commodity.String(), k2.Commodity.String())
}
