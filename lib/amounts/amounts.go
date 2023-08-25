package amounts

import (
	"regexp"
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/common/predicate"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/shopspring/decimal"
)

// Key represents a position.
type Key struct {
	Date           time.Time
	Account, Other *model.Account
	Commodity      *model.Commodity
	Valuation      *model.Commodity
	Description    string
}

func DateKey(date time.Time) Key {
	return Key{Date: date}
}

func DateCommodityKey(date time.Time, commodity *model.Commodity) Key {
	return Key{Date: date, Commodity: commodity}
}

func CommodityKey(commodity *model.Commodity) Key {
	return Key{Commodity: commodity}
}

func AccountKey(account *model.Account) Key {
	return Key{Account: account}
}

func AccountCommodityKey(account *model.Account, commodity *model.Commodity) Key {
	return Key{Account: account, Commodity: commodity}
}

// Amounts keeps track of amounts by account and commodity.
type Amounts map[Key]decimal.Decimal

// Amount returns the amount for the given key.
func (am Amounts) Amount(key Key) decimal.Decimal {
	return am[key]
}

func (am Amounts) Add(key Key, value decimal.Decimal) {
	am[key] = am[key].Add(value)
}

// Clone clones these amounts.
func (am Amounts) Clone() Amounts {
	clone := make(Amounts)
	for key, value := range am {
		clone[key] = value
	}
	return clone
}

// Minus mutably subtracts.
func (am Amounts) Minus(other Amounts) {
	for key, value := range other {
		am[key] = am[key].Sub(value)
	}
}

// Plus mutably adds.
func (am Amounts) Plus(other Amounts) {
	for key, value := range other {
		am[key] = am[key].Add(value)
	}
}

func (am Amounts) Index(cmp compare.Compare[Key]) []Key {
	index := make([]Key, 0, len(am))
	for k := range am {
		index = append(index, k)
	}
	if cmp != nil {
		compare.Sort(index, cmp)
	}
	return index
}

func (am Amounts) Commodities() set.Set[*model.Commodity] {
	commodities := set.New[*model.Commodity]()
	for key := range am {
		commodities.Add(key.Commodity)
	}
	return commodities
}

func (am Amounts) CommoditiesSorted() []*model.Commodity {
	commodities := am.Commodities()
	return dict.SortedKeys(commodities, commodity.Compare)
}

func (am Amounts) Dates() set.Set[time.Time] {
	res := set.New[time.Time]()
	for k := range am {
		res.Add(k.Date)
	}
	return res
}

func (am Amounts) DatesSorted() []time.Time {
	dates := am.Dates()
	return dict.SortedKeys(dates, compare.Time)
}

func (am Amounts) SumBy(pred func(k Key) bool, mapr func(k Key) Key) Amounts {
	res := make(Amounts)
	am.SumIntoBy(res, pred, mapr)
	return res
}

func (am Amounts) SumIntoBy(dest Amounts, pred func(k Key) bool, mapr func(k Key) Key) {
	if pred == nil {
		pred = predicate.True[Key]
	}
	if mapr == nil {
		mapr = mapper.Identity[Key]
	}
	for key, value := range am {
		if !pred(key) {
			continue
		}
		mappedKey := mapr(key)
		dest[mappedKey] = dest[mappedKey].Add(value)
	}
	for key, value := range dest {
		if value.IsZero() {
			delete(dest, key)
		}
	}
}

func (am Amounts) SumOver(pred func(k Key) bool) decimal.Decimal {
	var res decimal.Decimal
	for key, value := range am {
		if !pred(key) {
			continue
		}
		res = res.Add(value)
	}
	return res
}

type KeyMapper struct {
	Date                 mapper.Mapper[time.Time]
	Account, Other       mapper.Mapper[*model.Account]
	Commodity, Valuation mapper.Mapper[*model.Commodity]
	Description          mapper.Mapper[string]
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

func FilterDates(pred predicate.Predicate[time.Time]) predicate.Predicate[Key] {
	return func(k Key) bool { return pred(k.Date) }
}

func FilterCommodity(regexes []*regexp.Regexp) predicate.Predicate[Key] {
	if len(regexes) == 0 {
		return predicate.True[Key]
	}
	f := predicate.ByName[*model.Commodity](regexes)
	return func(k Key) bool {
		return f(k.Commodity)
	}
}

func FilterAccount(regexes []*regexp.Regexp) predicate.Predicate[Key] {
	if regexes == nil {
		return predicate.True[Key]
	}
	pred := predicate.ByName[*model.Account](regexes)
	return func(k Key) bool {
		return pred(k.Account)
	}
}

func FilterOther(regexes []*regexp.Regexp) predicate.Predicate[Key] {
	if regexes == nil {
		return predicate.True[Key]
	}
	pred := predicate.ByName[*model.Account](regexes)
	return func(k Key) bool {
		return pred(k.Other)
	}
}
