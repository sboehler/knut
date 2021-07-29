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

var (
	commodities = make(map[string]*Commodity)
	mutex       sync.RWMutex
)

// Commodity represents a currency or security.
type Commodity struct {
	name string
}

func get(name string) (*Commodity, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	c, ok := commodities[name]
	return c, ok
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

func create(name string) (*Commodity, error) {
	mutex.Lock()
	defer mutex.Unlock()
	// check if the commodity has been created in the meantime
	if c, ok := commodities[name]; ok {
		return c, nil
	}
	if !isValidCommodity(name) {
		return nil, fmt.Errorf("invalid commodity name %q", name)
	}
	var c = &Commodity{name}
	commodities[name] = c
	return c, nil
}

// Get creates a new commodity.
func Get(name string) (*Commodity, error) {
	if c, ok := get(name); ok {
		return c, nil
	}
	return create(name)
}

func (c Commodity) String() string {
	return c.name
}

// MarshalJSON marshals a commodity to JSON.
func (c Commodity) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.name)
}
