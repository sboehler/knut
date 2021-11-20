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

package ledger

import (
	"encoding/json"
	"fmt"
	"sync"
	"unicode"
)

// Commodity represents a currency or security.
type Commodity struct {
	name string
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
	commodities map[string]*Commodity
	mutex       sync.RWMutex
}

// NewCommodities creates a new thread-safe collection of commodities.
func NewCommodities() *Commodities {
	return &Commodities{
		commodities: make(map[string]*Commodity),
	}
}

// Get creates a new commodity.
func (c *Commodities) Get(name string) (*Commodity, error) {
	c.mutex.RLock()
	res, ok := c.commodities[name]
	c.mutex.RUnlock()
	if !ok {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		// check if the commodity has been created in the meantime
		if res, ok = c.commodities[name]; ok {
			return res, nil
		}
		if !isValidCommodity(name) {
			return nil, fmt.Errorf("invalid commodity name %q", name)
		}
		res = &Commodity{name}
		c.commodities[name] = res
	}
	return res, nil
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
