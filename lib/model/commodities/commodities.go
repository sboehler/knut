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

import "sync"

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

func create(name string) *Commodity {
	mutex.Lock()
	defer mutex.Unlock()
	if c, ok := commodities[name]; ok {
		return c
	}
	var c = &Commodity{name}
	commodities[name] = c
	return c
}

// Get creates a new commodity.
func Get(name string) *Commodity {
	if c, ok := get(name); ok {
		return c
	}
	return create(name)
}

func (c Commodity) String() string {
	return c.name
}
