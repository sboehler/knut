package amounts

import (
	"regexp"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
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

type Mapper func(Key) Key

type KeyMapper struct {
	Date                 func(time.Time) time.Time
	Account, Other       func(*journal.Account) *journal.Account
	Commodity, Valuation func(*journal.Commodity) *journal.Commodity
}

func (km KeyMapper) Build() Mapper {
	return func(k Key) Key {
		if km.Date == nil {
			k.Date = time.Time{}
		} else {
			k.Date = km.Date(k.Date)
		}
		if km.Account == nil {
			k.Account = nil
		} else {
			k.Account = km.Account(k.Account)
		}
		if km.Other == nil {
			k.Other = nil
		} else {
			k.Other = km.Other(k.Other)
		}
		if km.Commodity == nil {
			k.Commodity = nil
		} else {
			k.Commodity = km.Commodity(k.Commodity)
		}
		if km.Valuation == nil {
			k.Valuation = nil
		} else {
			k.Valuation = km.Valuation(k.Valuation)
		}
		return k
	}
}

func DefaultMapper(k Key) Key {
	return k
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
