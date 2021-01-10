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
	amount, value decimal.Decimal
}

// New creates a new amount with value zero.
func New(amount, value decimal.Decimal) Amount {
	return Amount{amount, value}
}

// Amount returns the amount.
func (a Amount) Amount() decimal.Decimal {
	return a.amount
}

// Value returns the value.
func (a Amount) Value() decimal.Decimal {
	return a.value
}

func (a Amount) String() string {
	return fmt.Sprintf("%s [%s]", a.amount, a.value)
}

// Plus computes the sum of two amounts.
func (a Amount) Plus(b Amount) Amount {
	return Amount{a.amount.Add(b.amount), a.value.Add(b.value)}
}

// Minus computes the difference between two amounts.
func (a Amount) Minus(b Amount) Amount {
	return Amount{a.amount.Sub(b.amount), a.value.Sub(b.value)}
}

// Neg computes the negation of this amount.
func (a Amount) Neg() Amount {
	return Amount{a.amount.Neg(), a.value.Neg()}
}

// Equal tests if both amounts are equal.
func (a Amount) Equal(b Amount) bool {
	return a.amount.Equal(b.amount) && a.value.Equal(b.value)
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

// Neg mutably negates the given vector.
func (v Vec) Neg() {
	for i := range v.Values {
		v.Values[i] = v.Values[i].Neg()
	}
}
