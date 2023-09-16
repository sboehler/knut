// Copyright 2021 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package price

import (
	"fmt"

	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/shopspring/decimal"
)

// Prices stores the price for a commodity to a target commodity
// Outer map: target commodity
// Inner map: commodity
// value: price in (target commodity / commodity)
type Prices map[*commodity.Commodity]NormalizedPrices

var one = decimal.NewFromInt(1)

// Insert inserts a new price.
func (ps Prices) Insert(commodity *commodity.Commodity, price decimal.Decimal, target *commodity.Commodity) {
	ps.addPrice(target, commodity, price)
	ps.addPrice(commodity, target, one.Div(price).Truncate(8))
}

func (ps Prices) addPrice(target, commodity *commodity.Commodity, price decimal.Decimal) {
	dict.GetDefault(ps, target, NewNormalizedPrices)[commodity] = price
}

// Normalize creates a normalized price map for the given commodity.
func (ps Prices) Normalize(t *commodity.Commodity) NormalizedPrices {
	res := NormalizedPrices{t: one}
	ps.normalize(t, res)
	return res
}

// normalize recursively computes prices by traversing the price graph.
// res must already contain a price for c.
func (ps Prices) normalize(c *commodity.Commodity, res NormalizedPrices) {
	for neighbor, price := range ps[c] {
		if _, done := res[neighbor]; done {
			continue
		}
		res[neighbor] = Multiply(price, res[c])
		ps.normalize(neighbor, res)
	}
}

// NormalizedPrices is a map representing the price of
// commodities in some base commodity.
type NormalizedPrices map[*commodity.Commodity]decimal.Decimal

func NewNormalizedPrices() NormalizedPrices {
	return make(NormalizedPrices)
}

func (np NormalizedPrices) Price(c *commodity.Commodity) (decimal.Decimal, error) {
	price, ok := np[c]
	if !ok {
		return decimal.Zero, fmt.Errorf("no price found for %v in %v", c, np)
	}
	return price, nil
}

// Valuate valuates the given amount.
func (np NormalizedPrices) Valuate(c *commodity.Commodity, a decimal.Decimal) (decimal.Decimal, error) {
	price, ok := np[c]
	if !ok {
		return decimal.Zero, fmt.Errorf("no price found for %v in %v", c, np)
	}
	return Multiply(a, price), nil
}

func Multiply(n1, n2 decimal.Decimal) decimal.Decimal {
	return n1.Mul(n2).Truncate(8)
}
