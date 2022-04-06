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

package journal

import (
	"regexp"
)

// Filter represents a filter for commodities and accounts.
type Filter struct {
	Accounts, Commodities *regexp.Regexp
}

// MatchAccount returns whether this filter matches the given account.
func (b Filter) MatchAccount(a *Account) bool {
	return b.Accounts == nil || b.Accounts.MatchString(a.String())
}

// MatchCommodity returns whether this filter matches the given commodity.
func (b Filter) MatchCommodity(c *Commodity) bool {
	return b.Commodities == nil || b.Commodities.MatchString(c.String())
}
