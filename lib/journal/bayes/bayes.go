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

package bayes

import (
	"math"
	"strings"

	"github.com/sboehler/knut/lib/common/dict"
	"github.com/sboehler/knut/lib/common/set"
	"github.com/sboehler/knut/lib/journal"
)

// Model implements a Bayes model for accounts and text tokens derived from transactions.
type Model struct {
	count                  int
	countByAccount         countByAccount
	countByTokenAndAccount map[string]countByAccount

	exclude *journal.Account
}

// NewModel creates a new model.
func NewModel(exclude *journal.Account) *Model {
	return &Model{
		count:                  0,
		countByAccount:         newCountByAccount(),
		countByTokenAndAccount: make(map[string]countByAccount),
	}
}

// Update updates the model with the given transaction.
func (m *Model) Update(t *journal.Transaction) {
	for _, p := range t.Postings {
		if p.Account == m.exclude {
			continue
		}
		m.count++
		m.countByAccount[p.Account]++
		for token := range tokenize(t.Description, p) {
			dict.GetDefault(m.countByTokenAndAccount, token, newCountByAccount)[p.Account]++
		}
	}
}

type countByAccount map[*journal.Account]int

func newCountByAccount() countByAccount {
	return make(map[*journal.Account]int)
}

// Infer replaces the given account with an inferred account.
// P(A | T1 & T2 & ... & Tn) ~ P(A) * P(T1|A) * P(T2|A) * ... * P(Tn|A)
func (m *Model) Infer(t *journal.Transaction, tbd *journal.Account) {
	def := math.Log(1.0 / float64(m.count))
	for _, posting := range t.Postings {
		if posting.Account != tbd {
			continue
		}
		scores := make(map[*journal.Account]float64)
		tokens := tokenize(t.Description, posting)
		for a, total := range m.countByAccount {
			if a == tbd || a == posting.Other {
				// ignore both TBD and the other account of this posting
				continue
			}
			scores[a] = math.Log(float64(total) / float64(m.count))
			for token := range tokens {
				if countForToken, ok := m.countByTokenAndAccount[token][a]; ok {
					scores[a] += math.Log(float64(countForToken) / float64(total))
				} else {
					// assign a low but positive default probability
					scores[a] += def
				}
			}
		}
		var selected *journal.Account
		max := math.Inf(-1)
		for a, score := range scores {
			if score > max {
				selected = a
				max = score
			}
		}
		posting.Account = selected
	}
}

func tokenize(desc string, posting *journal.Posting) set.Set[string] {
	tokens := append(strings.Fields(desc), posting.Commodity.Name(), posting.Amount.String(), posting.Other.Name())
	result := set.New[string]()
	for _, token := range tokens {
		result.Add(strings.ToLower(token))
	}
	return result
}
