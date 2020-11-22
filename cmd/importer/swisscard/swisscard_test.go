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

package swisscard

import (
	"github.com/sboehler/knut/cmd/cmdtest"
	"fmt"
	"testing"

	"github.com/sebdah/goldie/v2"
)

func TestGolden(t *testing.T) {
	tests := []string{
		"example1",
	}
	for _, test := range tests {
		test := test
		t.Run(test, func(t *testing.T) {
			t.Parallel()
			g := goldie.New(t)
			args := []string{"--account", "Liabilities:CreditCard", fmt.Sprintf("testdata/%s.input", test)}
			got := cmdtest.Run(t, CreateCmd(), args)
			g.Assert(t, test, got)
		})
	}
}
