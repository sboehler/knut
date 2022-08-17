package amounts2

import (
	"sort"
	"time"

	"github.com/sboehler/knut/lib/common"
	"github.com/sboehler/knut/lib/journal"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Account     *journal.Account
	Commodity   *journal.Commodity
	Date        time.Time
	Valuation   *journal.Commodity
	Description string
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

// Less establishes a partial common.ordering of keys.
func (p Key) Less(p1 Key) bool {
	if p.Account != p1.Account {
		return p.Account.Less(p1.Account)
	}
	return p.Commodity.String() < p1.Commodity.String()
}

// Amounts keeps track of amounts by account and commodity.
type Amounts map[Key]decimal.Decimal

// Amount returns the amount for the given account and commodity.
func (am Amounts) Amount(a *journal.Account, c *journal.Commodity) decimal.Decimal {
	return am[Key{Account: a, Commodity: c}]
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

type Frame struct {
	data map[Key]decimal.Decimal
}

func NewFrame(n int) *Frame {
	return &Frame{
		data: make(map[Key]decimal.Decimal),
	}
}

func (f *Frame) Get(k Key) (decimal.Decimal, bool) {
	d, ok := f.data[k]
	return d, ok
}

func (f *Frame) Add(k Key, v decimal.Decimal) {
	f.data[k] = f.data[k].Add(v)
}

func (f *Frame) Index(srt Sorter) []Key {
	res := make([]Key, 0, len(f.data))
	for k := range f.data {
		res = append(res, k)
	}
	srt.Sort(res)
	return res
}

func ByAccount(k1, k2 Key) common.Ordering {
	switch {
	case k1.Account == k2.Account:
		return common.Equal
	case k1.Account.Name() < k2.Account.Name():
		return common.Smaller
	default:
		return common.Greater
	}
}

func ByCommodity(k1, k2 Key) common.Ordering {
	switch {
	case k1.Commodity == k2.Commodity:
		return common.Equal
	case k1.Commodity.String() < k2.Commodity.String():
		return common.Smaller
	default:
		return common.Greater
	}
}

type Compare func(Key, Key) common.Ordering

type Sorter []Compare

func (s Sorter) Sort(ks []Key) {
	sort.Slice(ks, func(i, j int) bool {
		return s.Compare(ks[i], ks[j]) == common.Smaller
	})
}

func (s Sorter) Compare(k1, k2 Key) common.Ordering {
	for _, ord := range s {
		o := ord(k1, k2)
		if o == common.Equal {
			continue
		}
		return o
	}
	return common.Equal
}
