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

package report

import (
	"github.com/sboehler/knut/lib/model/commodities"
	"github.com/sboehler/knut/lib/vector"
)

// Segment is a hierarchical segment of a report.
type Segment struct {
	Key         string
	Positions   map[*commodities.Commodity]vector.Vector
	Subsegments []*Segment
}

// NewSegment creates a new segment.
func NewSegment(k string) *Segment {
	return &Segment{
		Key:         k,
		Positions:   make(map[*commodities.Commodity]vector.Vector),
		Subsegments: nil,
	}
}

func (s *Segment) insert(keys []string, pos Position) {
	if len(keys) > 0 {
		var (
			key        = keys[0]
			subsegment *Segment
		)
		for _, ss := range s.Subsegments {
			if ss.Key == key {
				subsegment = ss
				break
			}
		}
		if subsegment == nil {
			subsegment = NewSegment(key)
			s.Subsegments = append(s.Subsegments, subsegment)
		}
		subsegment.insert(keys[1:], pos)
	} else {
		if existing, ok := s.Positions[pos.Commodity]; ok {
			existing.Add(pos.Amounts)
		} else {
			s.Positions[pos.Commodity] = pos.Amounts
		}
	}
}

func (s *Segment) sum(m map[*commodities.Commodity]vector.Vector) map[*commodities.Commodity]vector.Vector {
	if m == nil {
		m = make(map[*commodities.Commodity]vector.Vector)
	}
	for _, ss := range s.Subsegments {
		ss.sum(m)
	}
	for c, a := range s.Positions {
		if _, ok := m[c]; !ok {
			m[c] = vector.New(len(a.Values))
		}
		m[c].Add(a)
	}
	return m
}
