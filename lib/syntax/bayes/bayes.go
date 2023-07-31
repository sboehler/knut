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
	"github.com/sboehler/knut/lib/syntax"
)

// Model implements a Bayes model for accounts and text tokens derived from transactions.
type Model struct {
	count                  int
	countByAccount         countByAccount
	countByTokenAndAccount map[token]countByAccount

	account string
}

type token string

// NewModel creates a new model.
func NewModel(account string) *Model {
	return &Model{
		count:                  0,
		countByAccount:         make(countByAccount),
		countByTokenAndAccount: make(map[token]countByAccount),
		account:                account,
	}
}

// Update updates the model with the given transaction.
func (m *Model) Update(t *syntax.Transaction) {
	for i, b := range t.Bookings {
		if b.Credit.Macro || b.Debit.Macro {
			continue
		}
		credit := b.Credit.Extract()
		debit := b.Debit.Extract()
		if credit == "" || debit == "" {
			continue
		}
		if credit == m.account || debit == m.account {
			continue
		}
		m.update(t, &t.Bookings[i], credit, debit)
		m.update(t, &t.Bookings[i], debit, credit)
	}
}

func (m *Model) update(t *syntax.Transaction, b *syntax.Booking, account, other string) {
	m.count++
	m.countByAccount[account]++
	for token := range tokenize(t, b, other) {
		dict.GetDefault(m.countByTokenAndAccount, token, newCountByAccount)[account]++
	}
}

type countByAccount map[string]int

func newCountByAccount() countByAccount {
	return make(map[string]int)
}

// Infer replaces the given account with an inferred account.
// P(A | T1 & T2 & ... & Tn) ~ P(A) * P(T1|A) * P(T2|A) * ... * P(Tn|A)
func (m *Model) Infer(t *syntax.Transaction) {
	for i := range t.Bookings {
		credit := t.Bookings[i].Credit.Extract()
		debit := t.Bookings[i].Debit.Extract()
		if credit == m.account {
			t.Bookings[i].Credit = m.inferAccount(t, &t.Bookings[i], debit)
		}
		if debit == m.account {
			t.Bookings[i].Debit = m.inferAccount(t, &t.Bookings[i], credit)
		}
	}
}

func (m *Model) inferAccount(t *syntax.Transaction, b *syntax.Booking, other string) syntax.Account {
	var (
		tokens = tokenize(t, b, other)
		max    = math.Inf(-1)
		best   string
	)
	for candidate := range m.countByAccount {
		if candidate == other {
			continue // the other account of this booking is not a valid candidate
		}
		score := m.scoreCandidate(candidate, tokens)
		if score > max {
			best = candidate
			max = score
		}
	}
	return syntax.Account{
		Range: syntax.Range{Start: 0, End: len(best), Text: best},
	}
}

func (m *Model) scoreCandidate(candidate string, tokens set.Set[token]) float64 {
	count := float64(m.countByAccount[candidate])
	score := math.Log(count / float64(m.count))
	for token := range tokens {
		if countForToken, ok := m.countByTokenAndAccount[token][candidate]; ok {
			score += math.Log(float64(countForToken) / count)
		} else {
			score += math.Log(1.0 / float64(m.count))
		}
	}
	return score
}

func tokenize(t *syntax.Transaction, b *syntax.Booking, other string) set.Set[token] {
	tokens := append(strings.Fields(t.Description.Content.Extract()), b.Commodity.Extract(), b.Amount.Extract(), other)
	result := set.New[token]()
	for _, t := range tokens {
		result.Add(token(strings.ToLower(t)))
	}
	return result
}
