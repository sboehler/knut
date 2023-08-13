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

package commodity

import (
	"fmt"
	"sync"
	"unicode"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/mapper"
	"github.com/sboehler/knut/lib/syntax"
)

// Registry is a thread-safe collection of commodities.
type Registry struct {
	index map[string]*Commodity
	mutex sync.RWMutex
}

// NewCommodities creates a new thread-safe collection of commodities.
func NewCommodities() *Registry {
	return &Registry{
		index: make(map[string]*Commodity),
	}
}

// Get creates a new commodity.
func (cs *Registry) Get(name string) (*Commodity, error) {
	cs.mutex.RLock()
	res, ok := cs.index[name]
	cs.mutex.RUnlock()
	if ok {
		return res, nil
	}
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	// check if the commodity has been created in the meantime
	if res, ok = cs.index[name]; ok {
		return res, nil
	}
	if !isValidCommodity(name) {
		return nil, fmt.Errorf("invalid commodity name %q", name)
	}
	res = &Commodity{name: name}
	cs.insert(res)

	return res, nil
}

func (as *Registry) Create(a syntax.Commodity) (*Commodity, error) {
	return as.Get(a.Extract())
}

func (cs *Registry) insert(c *Commodity) {
	cs.index[c.name] = c
}

// TagCurrency tags the commodity as a currency.
func (cs *Registry) TagCurrency(name string) error {
	commodity, err := cs.Get(name)
	if err != nil {
		return err
	}
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	commodity.IsCurrency = true
	return nil
}

func isValidCommodity(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !(unicode.IsLetter(c) || unicode.IsDigit(c)) {
			return false
		}
	}
	return true
}

func Map(t bool) func(*Commodity) *Commodity {
	if t {
		return mapper.Identity[*Commodity]
	}
	return mapper.Nil[*Commodity, Commodity]
}

func Compare(c1, c2 *Commodity) compare.Order {
	return compare.Ordered(c1.Name(), c2.Name())
}
