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

	"github.com/sboehler/knut/lib/common/filter"
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

// Period represents a time period
type Period struct {
	Start, End time.Time
}

// Less defines an ordering for periods.
func (p Period) Less(p2 Period) bool {
	return p.End.Before(p2.End)
}

// Periods returns a series of periods in the given interval,
// which contains both t0 and t1.
func Periods(t0, t1 time.Time, p Interval) []Period {
	var res []Period
	if p == Once {
		if t0.Before(t1) {
			res = append(res, Period{t0, t1})
		}
	} else {
		for t := t0; !t.After(t1); t = EndOf(t, p).AddDate(0, 0, 1) {
			ed := EndOf(t, p)
			if ed.After(t1) {
				ed = t1
			}
			res = append(res, Period{StartOf(t, p), ed})
		}
	}
	return res
}

// Contains returns whether the period contains the given
func (p Period) Contains(t time.Time) bool {
	return !p.Start.After(t) && !p.End.Before(t)
}

// Periods returns a series of periods in the given interval,
// which contains both t0 and t1.
func PeriodsN(t0, t1 time.Time, p Interval, n int) []Period {
	var res []Period
	if p == Once {
		if t0.Before(t1) {
			res = append(res, Period{t0, t1})
		}
	} else {
		for t := t1; !t.Before(t0); t = StartOf(t, p).AddDate(0, 0, -1) {
			sd := StartOf(t, p)
			if sd.Before(t0) {
				sd = t0
			}
			res = append(res, Period{sd, t})
			if len(res) == n {
				break
			}
		}
		reverse(res)
	}
	return res
}

func reverse(ps []Period) {
	for i := 0; i < len(ps)/2; i++ {
		ps[i], ps[len(ps)-i-1] = ps[len(ps)-i-1], ps[i]
	}
}

func Map(part []time.Time) func(time.Time) time.Time {
	return func(t time.Time) time.Time {
		index := sort.Search(len(part), func(i int) bool {
			// t <= part[i]
			return !part[i].Before(t)
		})
		if index < len(part) {
			return part[index]
		}
		return time.Time{}
	}
}

func CreatePartition(t0, t1 time.Time, p Interval, n int) []time.Time {
	var res []time.Time
	if p == Once {
		if t0.Before(t1) {
			res = append(res, t1)
		}
	} else {
		for t := t0; !t.After(t1); t = EndOf(t, p).AddDate(0, 0, 1) {
			ed := EndOf(t, p)
			if ed.After(t1) {
				ed = t1
			}
			res = append(res, ed)
		}
	}
	if n > 0 && len(res) > n {
		res = res[len(res)-n:]
	}
	return res
}

func Between(t0, t1 time.Time) filter.Filter[time.Time] {
	return func(t time.Time) bool {
		return !t.Before(t0) && !t.After(t1)
	}
}
