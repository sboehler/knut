// Copyright 2020 Silvio Böhler
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

package amount

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Amount represents an amount in some commodity, along
// with a set of valuations.
type Amount struct {
	amount     decimal.Decimal
	valuations []decimal.Decimal
}

// New creates a new amount with value zero.
func New(amount decimal.Decimal, valuations []decimal.Decimal) Amount {
	return Amount{amount, valuations}
}

// Amount returns the amount.
func (a Amount) Amount() decimal.Decimal {
	return a.amount
}

// Valuation returns the valuation at position i. If the valuation
// does not exist, 0 is returned.
func (a Amount) Valuation(i int) decimal.Decimal {
	if 0 <= i && i < len(a.valuations) {
		return a.valuations[i]
	}
	return decimal.Zero
}

func (a Amount) String() string {
	return fmt.Sprintf("%s %v", a.amount, a.valuations)
}

// Plus computes the sum of two amounts.
func (a Amount) Plus(b Amount) Amount {
	if len(a.valuations) < len(b.valuations) {
		return b.Plus(a)
	}
	amount := a.amount.Add(b.amount)
	if len(b.valuations) == 0 {
		return Amount{amount, a.valuations}
	}
	valuations := make([]decimal.Decimal, 0, len(a.valuations))
	for i, v := range b.valuations {
		valuations = append(valuations, a.valuations[i].Add(v))
	}
	if len(a.valuations) > len(b.valuations) {
		valuations = append(valuations, a.valuations[len(b.valuations):len(a.valuations)]...)
	}
	return Amount{amount, valuations}
}

// Minus computes the difference between two amounts.
func (a Amount) Minus(b Amount) Amount {
	return a.Plus(b.Neg())
}

// Neg computes the negation of this amount.
func (a Amount) Neg() Amount {
	amount := a.amount.Neg()
	if len(a.valuations) == 0 {
		return Amount{amount, nil}
	}
	valuations := make([]decimal.Decimal, 0, len(a.valuations))
	for _, v := range a.valuations {
		valuations = append(valuations, v.Neg())
	}
	return Amount{amount, valuations}
}

// Equal tests if both amounts are equal.
func (a Amount) Equal(b Amount) bool {
	if len(a.valuations) < len(b.valuations) {
		return b.Equal(a)
	}
	if !a.amount.Equal(b.amount) {
		return false
	}
	for i, v := range a.valuations {
		if !v.Equal(b.Valuation(i)) {
			return false
		}
	}
	return true
}

// Vec is a vector of decimals.
type Vec struct {
	Values []decimal.Decimal
}

// NewVec creates a new vector.
func NewVec(n int) Vec {
	return Vec{
		Values: make([]decimal.Decimal, n),
	}
}

// Add mutably adds the given vector to the receiver.
func (v Vec) Add(u Vec) {
	for i, a := range u.Values {
		v.Values[i] = v.Values[i].Add(a)
	}
}
