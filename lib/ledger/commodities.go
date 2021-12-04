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
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"unicode"
)

// Commodity represents a currency or security.
type Commodity struct {
	name       string
	IsCurrency bool
}

func (c Commodity) String() string {
	return c.name
}

// MarshalJSON marshals a commodity to JSON.
func (c Commodity) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.name)
}

// Commodities is a thread-safe collection of commodities.
type Commodities struct {
	lookup      map[string]*Commodity
	commodities []*Commodity
	mutex       sync.RWMutex
}

// NewCommodities creates a new thread-safe collection of commodities.
func NewCommodities() *Commodities {
	return &Commodities{
		lookup: make(map[string]*Commodity),
	}
}

// Get creates a new commodity.
func (cs *Commodities) Get(name string) (*Commodity, error) {
	cs.mutex.RLock()
	res, ok := cs.lookup[name]
	cs.mutex.RUnlock()
	if ok {
		return res, nil
	}
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	// check if the commodity has been created in the meantime
	if res, ok = cs.lookup[name]; ok {
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
	index := sort.Search(len(cs.commodities), func(i int) bool { return cs.commodities[i].name >= c.name })
	if index != len(cs.commodities) && cs.commodities[index].name == c.name {
		return
	}
	cs.commodities = append(cs.commodities, nil)
	copy(cs.commodities[index+1:], cs.commodities[index:])
	cs.commodities[index] = c
	cs.lookup[c.name] = c
}

// Enumerate enumerates the commodities.
func (cs *Commodities) Enumerate() <-chan *Commodity {
	ch := make(chan *Commodity)
	go func() {
		defer close(ch)
		for _, c := range cs.commodities {
			ch <- c
		}
	}()
	return ch
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
