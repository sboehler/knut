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

package report

import (
	"regexp"
	"time"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/vector"
)

// Report is a balance report for a range of dates.
type Report struct {
	Dates       []time.Time
	Segments    map[ledger.AccountType]*Segment
	Commodities []*ledger.Commodity
	Positions   map[*ledger.Commodity]vector.Vector
}

// Position is a position.
type Position struct {
	Account   *ledger.Account
	Commodity *ledger.Commodity
	Amounts   vector.Vector
}

// Less establishes a partial ordering of commodity accounts.
func (p Position) Less(p1 Position) bool {
	if p.Account.Type() != p1.Account.Type() {
		return p.Account.Type() < p1.Account.Type()
	}
	if p.Account.String() != p1.Account.String() {
		return p.Account.String() < p1.Account.String()
	}
	return p.Commodity.String() < p1.Commodity.String()
}

// Collapse is a rule for collapsing (shortening) accounts.
type Collapse struct {
	Level int
	Regex *regexp.Regexp
}

// MatchAccount determines whether this Collapse matches the
// given Account.
func (c Collapse) MatchAccount(a *ledger.Account) bool {
	return c.Regex == nil || c.Regex.MatchString(a.String())
}
