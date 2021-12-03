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

package report

import (
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/vector"
)

// Segment is a hierarchical segment of a report.
type Segment struct {
	Key         string
	Positions   map[*ledger.Commodity]vector.Vector
	Subsegments []*Segment
}

// NewSegment creates a new segment.
func NewSegment(k string) *Segment {
	return &Segment{
		Key:         k,
		Positions:   make(map[*ledger.Commodity]vector.Vector),
		Subsegments: nil,
	}
}

func (s *Segment) insert(keys []string, pos Position) {
	if len(keys) > 0 {
		head, tail := keys[0], keys[1:]
		s.findOrCreateSubsegment(head).insert(tail, pos)
	} else {
		s.addPosition(pos.Commodity, pos.Amounts)
	}
}

func (s *Segment) findOrCreateSubsegment(key string) *Segment {
	for _, ss := range s.Subsegments {
		if ss.Key == key {
			return ss
		}
	}
	var subsegment = NewSegment(key)
	s.Subsegments = append(s.Subsegments, subsegment)
	return subsegment
}

func (s *Segment) addPosition(c *ledger.Commodity, v vector.Vector) {
	pos, ok := s.Positions[c]
	if !ok {
		pos = vector.New(len(v.Values))
		s.Positions[c] = pos
	}
	pos.Add(v)
}

func (s *Segment) sum(m map[*ledger.Commodity]vector.Vector) {
	for _, ss := range s.Subsegments {
		ss.sum(m)
	}
	for c, a := range s.Positions {
		if _, ok := m[c]; !ok {
			m[c] = vector.New(len(a.Values))
		}
		m[c].Add(a)
	}
}
