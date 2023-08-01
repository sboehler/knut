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

package model

import (
	"fmt"
	"sync"
	"unicode"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/common/mapper"
)

// Commodities is a thread-safe collection of commodities.
type Commodities struct {
	index map[string]*Commodity
	mutex sync.RWMutex
}

// NewCommodities creates a new thread-safe collection of commodities.
func NewCommodities() *Commodities {
	return &Commodities{
		index: make(map[string]*Commodity),
	}
}

// Get creates a new commodity.
func (cs *Commodities) Get(name string) (*Commodity, error) {
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

func (cs *Commodities) insert(c *Commodity) {
	cs.index[c.name] = c
}

// TagCurrency tags the commodity as a currency.
func (cs *Commodities) TagCurrency(name string) error {
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

func MapCommodity(t bool) func(*Commodity) *Commodity {
	if t {
		return mapper.Identity[*Commodity]
	}
	return mapper.Nil[*Commodity]
}

func CompareCommodities(c1, c2 *Commodity) compare.Order {
	return compare.Ordered(c1.Name(), c2.Name())
}
