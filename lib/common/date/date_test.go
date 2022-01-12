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
	"testing"
	"time"
)

func TestStartOf(t *testing.T) {
	var tests = []struct {
		date   time.Time
		result map[Interval]time.Time
	}{
		{
			date: Date(2020, 1, 1),
			result: map[Interval]time.Time{
				Weekly:    Date(2019, 12, 30),
				Monthly:   Date(2020, 1, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 1, 31),
			result: map[Interval]time.Time{
				Weekly:    Date(2020, 1, 27),
				Monthly:   Date(2020, 1, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 2, 1),
			result: map[Interval]time.Time{
				Weekly:    Date(2020, 1, 27),
				Monthly:   Date(2020, 2, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 6, 1),
			result: map[Interval]time.Time{
				Quarterly: Date(2020, 4, 1),
			},
		},
		{
			date: Date(2020, 12, 3),
			result: map[Interval]time.Time{
				Quarterly: Date(2020, 10, 1),
			},
		},
	}

	for _, test := range tests {
		for interval, result := range test.result {
			if got := StartOf(test.date, interval); got != result {
				t.Errorf("StartOf(%v, %v): Got %v, wanted %v", test.date, interval, got, result)
			}
		}
	}
}

func TestEndOf(t *testing.T) {
	var tests = []struct {
		date   time.Time
		result map[Interval]time.Time
	}{
		{
			date: Date(2020, 1, 1),
			result: map[Interval]time.Time{
				Weekly:    Date(2020, 1, 5),
				Monthly:   Date(2020, 1, 31),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 1, 31),
			result: map[Interval]time.Time{
				Weekly:    Date(2020, 2, 2),
				Monthly:   Date(2020, 1, 31),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 2, 1),
			result: map[Interval]time.Time{
				Weekly:    Date(2020, 2, 2),
				Monthly:   Date(2020, 2, 29),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 6, 1),
			result: map[Interval]time.Time{
				Quarterly: Date(2020, 6, 30),
			},
		},
		{
			date: Date(2020, 12, 31),
			result: map[Interval]time.Time{
				Quarterly: Date(2020, 12, 31),
			},
		},
	}

	for _, test := range tests {
		for interval, result := range test.result {
			if got := EndOf(test.date, interval); got != result {
				t.Errorf("EndOf(%v, %v): Got %v, wanted %v", test.date, interval, got, result)
			}
		}
	}
}

func TestSeries(t *testing.T) {
	var tests = []struct {
		t0       time.Time
		t1       time.Time
		interval Interval
		result   []time.Time
	}{
		{
			t0:       Date(2020, 5, 19),
			t1:       Date(2020, 5, 22),
			interval: Once,
			result: []time.Time{
				Date(2020, 5, 19),
				Date(2020, 5, 22),
			},
		},
		{
			t0:       Date(2020, 5, 19),
			t1:       Date(2020, 5, 22),
			interval: Daily,
			result: []time.Time{
				Date(2020, 5, 18),
				Date(2020, 5, 19),
				Date(2020, 5, 20),
				Date(2020, 5, 21),
				Date(2020, 5, 22),
			},
		},
		{
			t0:       Date(2020, 1, 1),
			t1:       Date(2020, 1, 31),
			interval: Weekly,
			result: []time.Time{
				Date(2019, 12, 29),
				Date(2020, 1, 5),
				Date(2020, 1, 12),
				Date(2020, 1, 19),
				Date(2020, 1, 26),
				Date(2020, 2, 2),
			},
		},
		{
			t0:       Date(2019, 12, 31),
			t1:       Date(2020, 1, 31),
			interval: Monthly,
			result: []time.Time{
				Date(2019, 11, 30),
				Date(2019, 12, 31),
				Date(2020, 1, 31),
			},
		},
		{
			t0:       Date(2020, 1, 1),
			t1:       Date(2020, 1, 31),
			interval: Monthly,
			result: []time.Time{
				Date(2019, 12, 31),
				Date(2020, 1, 31),
			},
		},
		{
			t0:       Date(2017, 4, 1),
			t1:       Date(2019, 3, 3),
			interval: Yearly,
			result: []time.Time{
				Date(2016, 12, 31),
				Date(2017, 12, 31),
				Date(2018, 12, 31),
				Date(2019, 12, 31),
			},
		},
	}

	for _, test := range tests {
		if got := Series(test.t0, test.t1, test.interval); !Equal(got, test.result) {
			t.Errorf("Series(%v, %v, %v): Got %v, wanted %v", test.t0, test.t1, test.interval, got, test.result)
		}
	}
}

func Equal(d1, d2 []time.Time) bool {
	if len(d1) != len(d2) {
		return false
	}
	for i, d := range d1 {
		if d != d2[i] {
			return false
		}
	}
	return true
}
