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
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
)

// Model implements a Bayes model for accounts and text tokens derived from transactions.
type Model struct {
	total                  int
	totalByAccount         map[*journal.Account]int
	totalByTokenAndAccount map[string]map[*journal.Account]int

	exclude *journal.Account
}

// NewModel creates a new model.
func NewModel(exclude *journal.Account) *Model {
	return &Model{
		total:                  0,
		totalByAccount:         make(map[*journal.Account]int),
		totalByTokenAndAccount: make(map[string]map[*journal.Account]int),
	}
}

// Update updates the model with the given transaction.
func (m *Model) Update(t *ast.Transaction) {
	for _, p := range t.Postings {
		if p.Credit != m.exclude {
			m.total++
			m.totalByAccount[p.Credit]++
			for token := range tokenize(t.Description, &p, p.Credit) {
				tc := dict.GetDefault(m.totalByTokenAndAccount, token, func() map[*journal.Account]int { return make(map[*journal.Account]int) })
				tc[p.Credit]++
			}
		}
		if p.Debit != m.exclude {
			m.total++
			m.totalByAccount[p.Debit]++
			for token := range tokenize(t.Description, &p, p.Debit) {
				tc := dict.GetDefault(m.totalByTokenAndAccount, token, func() map[*journal.Account]int { return make(map[*journal.Account]int) })
				tc[p.Debit]++
			}
		}

	}
}

// Infer replaces the given account with an inferred account.
// P(A | T1 & T2 & ... & Tn) ~ P(A) * P(T1|A) * P(T2|A) * ... * P(Tn|A)
func (m *Model) Infer(t *ast.Transaction, tbd *journal.Account) {
	def := math.Log(1.0 / float64(m.total))
	for i := range t.Postings {
		posting := &t.Postings[i]
		if posting.Credit != tbd && posting.Debit != tbd {
			continue
		}
		scores := make(map[*journal.Account]float64)
		tokens := tokenize(t.Description, posting, tbd)
		for a, total := range m.totalByAccount {
			if a == posting.Credit || a == posting.Debit {
				// ignore both TBD and the other account of this posting
				continue
			}
			scores[a] = math.Log(float64(total) / float64(m.total))
			for token := range tokens {
				if countForToken, ok := m.totalByTokenAndAccount[token][a]; ok {
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
		if posting.Credit == tbd {
			posting.Credit = selected
		}
		if posting.Debit == tbd {
			posting.Debit = selected
		}
	}
}

func tokenize(desc string, posting *ast.Posting, account *journal.Account) map[string]struct{} {
	tokens := append(strings.Fields(desc), posting.Commodity.Name(), posting.Amount.String())
	if account == posting.Credit {
		tokens = append(tokens, "__knut_credit", posting.Debit.Name())
	}
	if account == posting.Debit {
		tokens = append(tokens, "__knut_debit", posting.Credit.Name())
	}
	result := make(map[string]struct{})
	for _, token := range tokens {
		result[strings.ToLower(token)] = struct{}{}
	}
	return result
}
