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

package commodities

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

// Commodities is a thread-safe collection of commodities.
type Commodities struct {
	commodities map[string]*Commodity
	mutex       sync.RWMutex
}

// New creates a new thread-safe collection of commodities.
func New() *Commodities {
	return &Commodities{
		commodities: make(map[string]*Commodity),
	}
}

func (c *Commodities) get(name string) (*Commodity, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	res, ok := c.commodities[name]
	return res, ok
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

func (c *Commodities) create(name string) (*Commodity, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check if the commodity has been created in the meantime
	if res, ok := c.commodities[name]; ok {
		return res, nil
	}
	if !isValidCommodity(name) {
		return nil, fmt.Errorf("invalid commodity name %q", name)
	}
	var res = &Commodity{name}
	c.commodities[name] = res
	return res, nil
}

// Get creates a new commodity.
func (c *Commodities) Get(name string) (*Commodity, error) {
	if c, ok := c.get(name); ok {
		return c, nil
	}
	return c.create(name)
}

func (c Commodity) String() string {
	return c.name
}

// MarshalJSON marshals a commodity to JSON.
func (c Commodity) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.name)
}

// TODO: remove
var cs = New()

// Get creates a new commodity.
func Get(name string) (*Commodity, error) {
	return cs.Get(name)
}
