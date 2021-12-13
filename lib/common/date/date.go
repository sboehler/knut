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

import "time"

// Period is a time interval.
type Period int

const (
	// Once represents the beginning of the interval.
	Once Period = iota
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

func (p Period) String() string {
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

// Date creates a new date.
func Date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// StartOf returns the first date in the given period which
// contains the receiver.
func StartOf(d time.Time, p Period) time.Time {
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
func EndOf(d time.Time, p Period) time.Time {
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

// Series returns a series of dates in the given interval,
// which contains both t0 and t1.
func Series(t0, t1 time.Time, p Period) []time.Time {
	if p == Once {
		return []time.Time{t0, t1}
	}
	var (
		res = []time.Time{StartOf(t0, p).AddDate(0, 0, -1)}
		t   = t0
	)
	for t == t1 || t.Before(t1) {
		res = append(res, EndOf(t, p))
		t = EndOf(t, p).AddDate(0, 0, 1)
	}
	return res
}

// Today returns today's date.
func Today() time.Time {
	now := time.Now()
	return Date(now.Year(), now.Month(), now.Day())
}
