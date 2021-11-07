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

package bayes

import (
	"math"
	"strings"

	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
)

// Model is a model trained from a journal
type Model struct {
	accounts      int
	accountCounts map[*accounts.Account]int
	tokenCounts   map[string]map[*accounts.Account]int
}

// NewModel creates a new model.
func NewModel() *Model {
	return &Model{
		accounts:      0,
		accountCounts: make(map[*accounts.Account]int),
		tokenCounts:   make(map[string]map[*accounts.Account]int),
	}
}

// Update updates the model with the given transaction.
func (m *Model) Update(t *ledger.Transaction) {
	for _, p := range t.Postings {
		m.accounts++
		m.accountCounts[p.Credit]++
		for _, token := range tokenize(t, p, p.Credit) {
			tc, ok := m.tokenCounts[token]
			if !ok {
				tc = make(map[*accounts.Account]int)
				m.tokenCounts[token] = tc
			}
			tc[p.Credit]++
		}
		m.accounts++
		m.accountCounts[p.Debit]++
		for _, token := range tokenize(t, p, p.Debit) {
			tc, ok := m.tokenCounts[token]
			if !ok {
				tc = make(map[*accounts.Account]int)
				m.tokenCounts[token] = tc
			}
			tc[p.Debit]++
		}

	}
}

// Infer replaces the given account with an inferred account.
func (m *Model) Infer(trx *ledger.Transaction, tbd *accounts.Account) {
	for _, posting := range trx.Postings {
		var tokens []string
		if posting.Credit == tbd {
			tokens = tokenize(trx, posting, posting.Credit)
		}
		if posting.Debit == tbd {
			tokens = tokenize(trx, posting, posting.Debit)
		}
		var scores = make(map[*accounts.Account]float64)
		for a, accountCount := range m.accountCounts {
			if a == tbd {
				continue
			}
			scores[a] = math.Log(float64(accountCount) / float64(m.accounts))
			for token := range dedup(tokens) {
				if tokenCount, ok := m.tokenCounts[token][a]; ok {
					scores[a] += math.Log(float64(tokenCount) / float64(accountCount))
				} else {
					// assign a low but positive default probability
					scores[a] += math.Log(1.0 / float64(m.accounts))
				}
			}
		}
		var (
			selected *accounts.Account
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

func dedup(ss []string) map[string]bool {
	var res = make(map[string]bool)
	for _, s := range ss {
		res[s] = true
	}
	return res
}

func tokenize(trx *ledger.Transaction, posting ledger.Posting, account *accounts.Account) []string {
	var tokens = append(strings.Fields(trx.Description), posting.Commodity.String(), posting.Amount.String())
	if account == posting.Credit {
		tokens = append(tokens, "credit", posting.Debit.String())
	}
	if account == posting.Debit {
		tokens = append(tokens, "debit", posting.Credit.String())
	}
	var result = make([]string, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, strings.ToLower(token))
	}
	return result
}
