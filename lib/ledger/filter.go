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

package ledger

import (
	"regexp"
)

// Filter represents a filter creating a
type Filter struct {
	Accounts, Commodities *regexp.Regexp
}

// MatchAccount returns whether this filterthe given Account.
func (b Filter) MatchAccount(a *Account) bool {
	return b.Accounts == nil || b.Accounts.MatchString(a.String())
}

// MatchCommodity returns whether this filter matches the given Commodity.
func (b Filter) MatchCommodity(c *Commodity) bool {
	return b.Commodities == nil || b.Commodities.MatchString(c.String())
}

// MatchPosting returns whether this filter matches the given Posting.
func (b Filter) MatchPosting(p Posting) bool {
	return (b.MatchAccount(p.Credit) || b.MatchAccount(p.Debit)) && b.MatchCommodity(p.Commodity)
}
