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

package prices

import (
	"fmt"

	"github.com/sboehler/knut/lib/ledger"

	"github.com/shopspring/decimal"
)

// Prices stores the price for a commodity to a target commodity
// Outer map: target commodity
// Inner map: commodity
// value: price in (target commodity / commodity)
type Prices map[*ledger.Commodity]map[*ledger.Commodity]decimal.Decimal

var one = decimal.NewFromInt(1)

// Insert inserts a new price.
func (p Prices) Insert(pr *ledger.Price) {
	p.addPrice(pr.Target, pr.Commodity, pr.Price)
	p.addPrice(pr.Commodity, pr.Target, one.Div(pr.Price).Truncate(8))
}

func (p Prices) addPrice(target, commodity *ledger.Commodity, pr decimal.Decimal) {
	i, ok := p[target]
	if !ok {
		i = make(map[*ledger.Commodity]decimal.Decimal)
		p[target] = i
	}
	i[commodity] = pr
}

// Normalize creates a normalized price map for the given commodity.
func (p Prices) Normalize(c *ledger.Commodity) NormalizedPrices {
	// prices in (target commodity / commodity)
	var (
		todo = NormalizedPrices{c: one}
		done = make(NormalizedPrices)

		currentC *ledger.Commodity
		currentP decimal.Decimal
	)
	for len(todo) > 0 {
		// we're interested in an arbitrary element of the map
		for currentC, currentP = range todo {
			break
		}
		done[currentC] = currentP
		for neighbor, price := range p[currentC] {
			if _, ok := done[neighbor]; ok {
				continue
			}
			todo[neighbor] = price.Mul(currentP).Truncate(8)
		}
		delete(todo, currentC)
	}
	return done
}

// NormalizedPrices is a map representing the price of
// commodities in some base commodity.
type NormalizedPrices map[*ledger.Commodity]decimal.Decimal

// Valuate valuates the given amount.
func (n NormalizedPrices) Valuate(c *ledger.Commodity, a decimal.Decimal) (decimal.Decimal, error) {
	price, ok := n[c]
	if !ok {
		return decimal.Zero, fmt.Errorf("no price found for %v in %v", c, n)
	}
	return a.Mul(price).Truncate(8), nil
}
