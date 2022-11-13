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

package date

import (
	"sort"
	"time"
)

// Interval is a time interval.
type Interval int

const (
	// Once represents the beginning of the interval.
	Once Interval = iota
	// Daily is a daily interval.
	Daily
	// Weekly is a weekly interval.
	Weekly
	// Monthly is a monthly interval.
	Monthly
	// Quarterly is a quarterly interval.
	Quarterly
	// Yearly is a yearly interval.
	Yearly
)

func (p Interval) String() string {
	switch p {
	case Once:
		return "once"
	case Daily:
		return "daily"
	case Weekly:
		return "weekly"
	case Monthly:
		return "monthly"
	case Quarterly:
		return "quarterly"
	case Yearly:
		return "yearly"
	}
	return ""
}

// Date creates a new
func Date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// StartOf returns the first date in the given period which
// contains the receiver.
func StartOf(d time.Time, p Interval) time.Time {
	switch p {
	case Once:
		return d
	case Daily:
		return d
	case Weekly:
		var x = (int(d.Weekday()) + 6) % 7
		return d.AddDate(0, 0, -x)
	case Monthly:
		return Date(d.Year(), d.Month(), 1)
	case Quarterly:
		return Date(d.Year(), ((d.Month()-1)/3*3)+1, 1)
	case Yearly:
		return Date(d.Year(), 1, 1)
	}
	return d
}

// EndOf returns the last date in the given period that contains
// the receiver.
func EndOf(d time.Time, p Interval) time.Time {
	switch p {
	case Once:
		return d
	case Daily:
		return d
	case Weekly:
		var x = (7 - int(d.Weekday())) % 7
		return d.AddDate(0, 0, x)
	case Monthly:
		return StartOf(d, Monthly).AddDate(0, 1, -1)
	case Quarterly:
		return StartOf(d, Quarterly).AddDate(0, 3, 0).AddDate(0, 0, -1)
	case Yearly:
		return Date(d.Year(), 12, 31)
	}

	return d
}

// Today returns today's
func Today() time.Time {
	now := time.Now().Local()
	return Date(now.Year(), now.Month(), now.Day())
}

// Partition is a partition of the timeline.
// Invariants:
// - t1 is always the last element of ends.
// - len(ends) > 0
type Partition struct {
	t0, t1 time.Time
	ends   []time.Time
}

func CreatePartition(t0, t1 time.Time, p Interval, n int) Partition {
	if p == Once {
		return Partition{
			t0:   t0,
			t1:   t1,
			ends: []time.Time{t1},
		}
	}
	var res []time.Time
	for t := t0; !t.After(t1); t = EndOf(t, p).AddDate(0, 0, 1) {
		ed := EndOf(t, p)
		if ed.After(t1) {
			ed = t1
		}
		res = append(res, ed)
	}
	if n > 0 && len(res) > n {
		res = res[len(res)-n:]
	}
	return Partition{
		t0:   t0,
		t1:   t1,
		ends: res,
	}
}

func (p Partition) MapToEndOfPeriod(t time.Time) time.Time {
	index := sort.Search(len(p.ends), func(i int) bool {
		// find first i where p.ends[i] >= t
		return !p.ends[i].Before(t)
	})
	if index < len(p.ends) {
		return p.ends[index]
	}
	return time.Time{}
}

func (p Partition) ClosingDates() []time.Time {
	var res []time.Time
	for _, d := range p.ends[:len(p.ends)-1] {
		res = append(res, d.AddDate(0, 0, 1))
	}
	return res
}

func (p Partition) EndDates() []time.Time {
	res := make([]time.Time, len(p.ends))
	copy(res, p.ends)
	return res
}

func (p *Partition) Contain(t time.Time) bool {
	return !t.Before(p.t0) && !t.After(p.t1)
}
