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

package vector

import (
	"github.com/shopspring/decimal"
)

// Vector is a vector of decimals.
type Vector struct {
	Values []decimal.Decimal
}

// New creates a new vector.
func New(n int) Vector {
	return Vector{
		Values: make([]decimal.Decimal, n),
	}
}

// Add mutably adds the given vector to the receiver.
func (v Vector) Add(u Vector) {
	for i, a := range u.Values {
		v.Values[i] = v.Values[i].Add(a)
	}
}

// Neg mutably negates the given vector.
func (v Vector) Neg() {
	for i := range v.Values {
		v.Values[i] = v.Values[i].Neg()
	}
}
