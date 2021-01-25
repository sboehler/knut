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
	"github.com/shopspring/decimal"
)

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
