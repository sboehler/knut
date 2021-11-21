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

package balance

import (
	"fmt"

	"github.com/sboehler/knut/lib/ledger"
)

// Accounts keeps track of open accounts.
type Accounts map[*ledger.Account]bool

// Open opens an account.
func (oa Accounts) Open(a *ledger.Account) error {
	if oa[a] {
		return fmt.Errorf("account %v is already open", a)
	}
	oa[a] = true
	return nil
}

// Close closes an account.
func (oa Accounts) Close(a *ledger.Account) error {
	if !oa[a] {
		return fmt.Errorf("account %v is already closed", a)
	}
	delete(oa, a)
	return nil
}

// IsOpen returns whether an account is open.
func (oa Accounts) IsOpen(a *ledger.Account) bool {
	if oa[a] {
		return true
	}
	return a.Type() == ledger.EQUITY
}

// Copy copies accounts.
func (oa Accounts) Copy() Accounts {
	var res = make(map[*ledger.Account]bool, len(oa))
	for a := range oa {
		res[a] = true
	}
	return res
}
