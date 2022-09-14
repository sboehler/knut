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

package journal

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Prices stores the price for a commodity to a target commodity
// Outer map: target commodity
// Inner map: commodity
// value: price in (target commodity / commodity)
type Prices map[*Commodity]map[*Commodity]decimal.Decimal

var one = decimal.NewFromInt(1)

// Insert inserts a new price.
func (p Prices) Insert(commodity *Commodity, price decimal.Decimal, target *Commodity) {
	p.addPrice(target, commodity, price)
	p.addPrice(commodity, target, one.Div(price).Truncate(8))
}

func (pr Prices) addPrice(target, commodity *Commodity, p decimal.Decimal) {
	i, ok := pr[target]
	if !ok {
		i = make(map[*Commodity]decimal.Decimal)
		pr[target] = i
	}
	i[commodity] = p
}

// Normalize creates a normalized price map for the given commodity.
func (pr Prices) Normalize(t *Commodity) NormalizedPrices {
	res := NormalizedPrices{t: one}
	pr.normalize(t, res)
	return res
}

// normalize recursively computes prices by traversing the price graph.
// res must already contain a price for c.
func (pr Prices) normalize(c *Commodity, res NormalizedPrices) {
	for neighbor, price := range pr[c] {
		if _, done := res[neighbor]; done {
			continue
		}
		p := price.Mul(res[c]).Truncate(8)
		res[neighbor] = p
		pr.normalize(neighbor, res)
	}
}

// NormalizedPrices is a map representing the price of
// commodities in some base commodity.
type NormalizedPrices map[*Commodity]decimal.Decimal

// Valuate valuates the given amount.
func (n NormalizedPrices) Valuate(c *Commodity, a decimal.Decimal) (decimal.Decimal, error) {
	price, ok := n[c]
	if !ok {
		return decimal.Zero, fmt.Errorf("no price found for %v in %v", c, n)
	}
	return a.Mul(price).Truncate(8), nil
}
