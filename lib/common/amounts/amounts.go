package amounts

import (
	"regexp"
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/common/order"
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

func (am Amounts) Index(less func(k1, k2 Key) bool) []Key {
	res := make([]Key, 0, len(am))
	for k := range am {
		res = append(res, k)
	}
	if less != nil {
		sort.Slice(res, func(i, j int) bool {
			return less(res[i], res[j])
		})
	}
	return res
}

type Mapper func(journal.Context, Key) Key

type KeyMapper struct {
	Date                 func(time.Time) time.Time
	Account, Other       func(journal.Context, *journal.Account) *journal.Account
	Commodity, Valuation func(journal.Context, *journal.Commodity) *journal.Commodity
}

func (km KeyMapper) Build() Mapper {
	return func(jctx journal.Context, k Key) Key {
		if km.Date == nil {
			k.Date = time.Time{}
		} else {
			k.Date = km.Date(k.Date)
		}
		if km.Account == nil {
			k.Account = nil
		} else {
			k.Account = km.Account(jctx, k.Account)
		}
		if km.Other == nil {
			k.Other = nil
		} else {
			k.Other = km.Other(jctx, k.Other)
		}
		if km.Commodity == nil {
			k.Commodity = nil
		} else {
			k.Commodity = km.Commodity(jctx, k.Commodity)
		}
		if km.Valuation == nil {
			k.Valuation = nil
		} else {
			k.Valuation = km.Valuation(jctx, k.Valuation)
		}
		return k
	}
}

func DefaultMapper(_ journal.Context, k Key) Key {
	return k
}

type MapDate struct {
	From, To time.Time
	Interval date.Interval
	Last     int
}

func (tp MapDate) Build() func(time.Time) time.Time {
	part := createPartition(tp.From, tp.To, tp.Interval, tp.Last)
	return func(t time.Time) time.Time {
		index := sort.Search(len(part), func(i int) bool {
			return !part[i].Before(t)
		})
		if index < len(part) {
			return part[index]
		}
		return time.Time{}
	}
}

func createPartition(t0, t1 time.Time, p date.Interval, n int) []time.Time {
	var res []time.Time
	if p == date.Once {
		if t0.Before(t1) {
			res = append(res, t1)
		}
	} else {
		for t := t0; !t.After(t1); t = date.EndOf(t, p).AddDate(0, 0, 1) {
			ed := date.EndOf(t, p)
			if ed.After(t1) {
				ed = t1
			}
			res = append(res, ed)
		}
	}
	if n > 0 && len(res) > n {
		res = res[len(res)-n:]
	}
	return res
}

func MapAccount(m journal.Mapping) func(journal.Context, *journal.Account) *journal.Account {
	return func(jctx journal.Context, a *journal.Account) *journal.Account {
		return jctx.Accounts().Map(a, m)
	}
}

func ShowCommodity(t bool) func(journal.Context, *journal.Commodity) *journal.Commodity {
	return func(jctx journal.Context, c *journal.Commodity) *journal.Commodity {
		if t {
			return c
		}
		return nil
	}
}

type KeyFilter func(Key) bool

func CombineKeyFilters(fs ...KeyFilter) KeyFilter {
	return func(k Key) bool {
		for _, f := range fs {
			if !f(k) {
				return false
			}
		}
		return true
	}
}

func DefaultKeyFilter(_ Key) bool {
	return true
}

func FilterCommodity(r *regexp.Regexp) KeyFilter {
	if r == nil {
		return DefaultKeyFilter
	}
	return func(k Key) bool {
		return r.MatchString(k.Commodity.String())
	}
}

func FilterAccount(r *regexp.Regexp) KeyFilter {
	if r == nil {
		return DefaultKeyFilter
	}
	return func(k Key) bool {
		return r.MatchString(k.Account.String())
	}
}

func FilterOther(r *regexp.Regexp) KeyFilter {
	if r == nil {
		return DefaultKeyFilter
	}
	return func(k Key) bool {
		return r.MatchString(k.Other.String())
	}
}

func SortByDate(k1, k2 Key) order.Ordering {
	return order.CompareTime(k1.Date, k2.Date)
}

func SortByAccount(jctx journal.Context, w map[*journal.Account]float64) order.Compare[Key] {
	s := journal.CompareWeighted(jctx, w)
	return func(k1, k2 Key) order.Ordering {
		return s(k1.Account, k2.Account)
	}
}

func SortByCommodity(k1, k2 Key) order.Ordering {
	return order.CompareOrdered(k1.Commodity.String(), k2.Commodity.String())
}
