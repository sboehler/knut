package amounts

import (
	"regexp"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/filter"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Date           time.Time
	Account, Other *journal.Account
	Commodity      *journal.Commodity
	Valuation      *journal.Commodity
	Description    string
}

func DateKey(d time.Time) Key {
	return Key{Date: d}
}

func DateCommodityKey(d time.Time, c *journal.Commodity) Key {
	return Key{Date: d, Commodity: c}
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

// Plus mutably adds.
func (am Amounts) Plus(a Amounts) Amounts {
	for ca, v := range a {
		am[ca] = am[ca].Add(v)
	}
	return am
}

func (am Amounts) Index(cmp compare.Compare[Key]) []Key {
	res := make([]Key, 0, len(am))
	for k := range am {
		res = append(res, k)
	}
	if cmp != nil {
		compare.Sort(res, cmp)
	}
	return res
}

// func (am Amounts) CumulativeSum(ds []time.Time) {
// 	idx := am.Index(func(k1, k2 Key) compare.Order {
// 		return compare.Time(k1.Date, k2.Date)
// 	})
// 	var previous, current time.Time
// 	keys := make(map[Key]struct{})
// 	for _, d := range ds {

// 	}
// 	for _, k := range idx {
// 		if current != k.Date {
// 			for kg := range keys {
// 				kPrev, kCur := kg, kg
// 				kPrev.Date = previous
// 				kCur.Date = current
// 				if !am[kPrev].IsZero() {
// 					am[kCur] = am[kCur].Add(am[kPrev])
// 				}
// 			}
// 			previous = current
// 			current = k.Date
// 			fmt.Println(previous, current)
// 		}
// 		gen := k
// 		gen.Date = time.Time{}
// 		keys[gen] = struct{}{}
// 	}
// 	for kg := range keys {
// 		kPrev, kCur := kg, kg
// 		kPrev.Date = previous
// 		kCur.Date = current
// 		if !am[kPrev].IsZero() {
// 			am[kCur] = am[kCur].Add(am[kPrev])
// 		}
// 	}
// }

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
	am.SumIntoBy(res, f, m)
	return res
}

func (am Amounts) SumIntoBy(as Amounts, f func(k Key) bool, m func(k Key) Key) {
	if f == nil {
		f = filter.AllowAll[Key]
	}
	if m == nil {
		m = mapper.Identity[Key]
	}
	for k, v := range am {
		if !f(k) {
			continue
		}
		kn := m(k)
		as[kn] = as[kn].Add(v)
	}
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

type KeyMapper struct {
	Date                 func(time.Time) time.Time
	Account, Other       func(*journal.Account) *journal.Account
	Commodity, Valuation func(*journal.Commodity) *journal.Commodity
	Description          func(string) string
}

func (km KeyMapper) Build() mapper.Mapper[Key] {
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
		if km.Description != nil {
			res.Description = km.Description(k.Description)
		}
		return res
	}
}

func FilterDates(t time.Time) filter.Filter[Key] {
	return func(k Key) bool {
		return !k.Date.After(t)
	}
}

func FilterCommodity(rx []*regexp.Regexp) filter.Filter[Key] {
	if len(rx) == 0 {
		return filter.AllowAll[Key]
	}
	f := filter.ByName[*journal.Commodity](rx)
	return func(k Key) bool {
		return f(k.Commodity)
	}
}

func FilterAccount(r []*regexp.Regexp) filter.Filter[Key] {
	if r == nil {
		return filter.AllowAll[Key]
	}
	f := filter.ByName[*journal.Account](r)
	return func(k Key) bool {
		return f(k.Account)
	}
}

func FilterOther(r []*regexp.Regexp) filter.Filter[Key] {
	if r == nil {
		return filter.AllowAll[Key]
	}
	f := filter.ByName[*journal.Account](r)
	return func(k Key) bool {
		return f(k.Other)
	}
}
