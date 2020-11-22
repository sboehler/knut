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
	"testing"

	"github.com/shopspring/decimal"
)

func TestPlus(t *testing.T) {
	for i := -10; i < 10; i++ {
		for j := -10; j < 10; j++ {
			iDec := decimal.NewFromInt(int64(i))
			jDec := decimal.NewFromInt(int64(j))
			wantDec := decimal.NewFromInt(int64(i + j))
			a := New(iDec, []decimal.Decimal{iDec, iDec})
			b := New(jDec, []decimal.Decimal{jDec, jDec})
			want := New(wantDec, []decimal.Decimal{wantDec, wantDec})
			if got := a.Plus(b); !got.Equal(want) {
				t.Errorf("%s + %v = %v, want %v", a, b, got, want)
			}
		}
	}
}

func TestMinus(t *testing.T) {
	for i := -10; i < 10; i++ {
		for j := -10; j < 10; j++ {
			iDec := decimal.NewFromInt(int64(i))
			jDec := decimal.NewFromInt(int64(j))
			wantDec := decimal.NewFromInt(int64(i - j))
			a := New(iDec, []decimal.Decimal{iDec, iDec})
			b := New(jDec, []decimal.Decimal{jDec, jDec})
			want := New(wantDec, []decimal.Decimal{wantDec, wantDec})
			if got := a.Minus(b); !got.Equal(want) {
				t.Errorf("%s + %v = %v, want %v", a, b, got, want)
			}
		}
	}
}
