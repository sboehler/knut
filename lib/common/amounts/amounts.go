package amounts

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Date      time.Time
	Account   *journal.Account
	Commodity *journal.Commodity
	Valuation *journal.Commodity
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

func (am Amounts) Index() []Key {
	res := make([]Key, 0, len(am))
	for k := range am {
		res = append(res, k)
	}
	return res
}

type Mapper func(Key) Key

func Combine(ms ...Mapper) Mapper {
	return func(k Key) Key {
		for _, m := range ms {
			k = m(k)
		}
		return k
	}
}

func NoDate() Mapper {
	return func(k Key) Key {
		k.Date = time.Time{}
		return k
	}
}

func TimePartition(t0, t1 time.Time, p date.Interval, n int) Mapper {
	part := createPartition(t0, t1, p, n)
	return func(k Key) Key {
		index := sort.Search(len(part), func(i int) bool {
			return !part[i].Before(k.Date)
		})
		if index == len(part) {
			k.Date = time.Time{}
		}
		k.Date = part[index]
		return k
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

func MapAccounts(jctx journal.Context, m journal.Mapping) Mapper {
	return func(k Key) Key {
		k.Account = jctx.Accounts().Map(k.Account, m)
		return k
	}
}
