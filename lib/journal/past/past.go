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

package past

import (
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// PAST is a processed AST.
type PAST struct {
	Days    []*Day
	Context journal.Context
}

// Day represents a day of activity in the processed AST.
type Day struct {
	AST          *ast.Day
	Date         time.Time
	Transactions []*ast.Transaction
	Amounts      amounts.Amounts
}

// Less establishes an ordering on Day.
func (d *Day) Less(d2 *Day) bool {
	return d.Date.Before(d2.Date)
}
