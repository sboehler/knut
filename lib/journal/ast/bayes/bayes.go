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

// Model is a model trained from a journal
type Model struct {
	accounts      int
	accountCounts map[*journal.Account]int
	tokenCounts   map[string]map[*journal.Account]int

	exclude *journal.Account
}

// NewModel creates a new model.
func NewModel(exclude *journal.Account) *Model {
	return &Model{
		accounts:      0,
		accountCounts: make(map[*journal.Account]int),
		tokenCounts:   make(map[string]map[*journal.Account]int),
	}
}

// Update updates the model with the given transaction.
func (m *Model) Update(t *ast.Transaction) {
	for _, p := range t.Postings {
		if p.Credit != m.exclude {
			m.accounts++
			m.accountCounts[p.Credit]++
			for token := range tokenize(t.Description, &p, p.Credit) {
				tc, ok := m.tokenCounts[token]
				if !ok {
					tc = make(map[*journal.Account]int)
					m.tokenCounts[token] = tc
				}
				tc[p.Credit]++
			}
		}
		if p.Debit != m.exclude {
			m.accounts++
			m.accountCounts[p.Debit]++
			for token := range tokenize(t.Description, &p, p.Debit) {
				tc := dict.GetDefault(m.tokenCounts, token, func() map[*journal.Account]int { return make(map[*journal.Account]int) })
				tc[p.Debit]++
			}
		}

	}
}

// Infer replaces the given account with an inferred account.
func (m *Model) Infer(t *ast.Transaction, tbd *journal.Account) {
	for i := range t.Postings {
		var (
			posting = &t.Postings[i]
			tokens  map[string]struct{}
		)
		switch tbd {
		case posting.Credit, posting.Debit:
			tokens = tokenize(t.Description, posting, tbd)
		case posting.Debit:
			tokens = tokenize(t.Description, posting, tbd)
		default:
			continue
		}
		scores := make(map[*journal.Account]float64)
		for a, accountCount := range m.accountCounts {
			if a == tbd {
				continue
			}
			scores[a] = math.Log(float64(accountCount) / float64(m.accounts))
			for token := range tokens {
				if tokenCount, ok := m.tokenCounts[token][a]; ok {
					scores[a] += math.Log(float64(tokenCount) / float64(accountCount))
				} else {
					// assign a low but positive default probability
					scores[a] += math.Log(1.0 / float64(m.accounts))
				}
			}
		}
		var (
			selected *journal.Account
			max      = math.Inf(-1)
		)
		for a, score := range scores {
			if score > max && a != posting.Credit && a != posting.Debit {
				selected = a
				max = score
			}
		}
		if selected != nil {
			if posting.Credit == tbd {
				posting.Credit = selected
			}
			if posting.Debit == tbd {
				posting.Debit = selected
			}
		}
	}
}

func tokenize(desc string, posting *ast.Posting, account *journal.Account) map[string]struct{} {
	tokens := append(strings.Fields(desc), posting.Commodity.Name(), posting.Amount.String())
	if account == posting.Credit {
		tokens = append(tokens, "credit", posting.Debit.String())
	}
	if account == posting.Debit {
		tokens = append(tokens, "debit", posting.Credit.String())
	}
	result := make(map[string]struct{})
	for _, token := range tokens {
		result[strings.ToLower(token)] = struct{}{}
	}
	return result
}
