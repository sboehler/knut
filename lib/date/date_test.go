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

package date

import (
	"testing"
	"time"
)

func TestStartOf(t *testing.T) {
	tests := []struct {
		date   time.Time
		result map[Period]time.Time
	}{
		{
			date: Date(2020, 1, 1),
			result: map[Period]time.Time{
				Weekly:    Date(2019, 12, 30),
				Monthly:   Date(2020, 1, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 1, 31),
			result: map[Period]time.Time{
				Weekly:    Date(2020, 1, 27),
				Monthly:   Date(2020, 1, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 2, 1),
			result: map[Period]time.Time{
				Weekly:    Date(2020, 1, 27),
				Monthly:   Date(2020, 2, 1),
				Quarterly: Date(2020, 1, 1),
			},
		},
		{
			date: Date(2020, 6, 1),
			result: map[Period]time.Time{
				Quarterly: Date(2020, 4, 1),
			},
		},
		{
			date: Date(2020, 12, 3),
			result: map[Period]time.Time{
				Quarterly: Date(2020, 10, 1),
			},
		},
	}

	for _, test := range tests {
		for period, result := range test.result {
			if got := StartOf(test.date, period); got != result {
				t.Errorf("StartOf(%v, %v): Got %v, wanted %v", test.date, period, got, result)
			}
		}
	}
}

func TestEndOf(t *testing.T) {
	tests := []struct {
		date   time.Time
		result map[Period]time.Time
	}{
		{
			date: Date(2020, 1, 1),
			result: map[Period]time.Time{
				Weekly:    Date(2020, 1, 5),
				Monthly:   Date(2020, 1, 31),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 1, 31),
			result: map[Period]time.Time{
				Weekly:    Date(2020, 2, 2),
				Monthly:   Date(2020, 1, 31),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 2, 1),
			result: map[Period]time.Time{
				Weekly:    Date(2020, 2, 2),
				Monthly:   Date(2020, 2, 29),
				Quarterly: Date(2020, 3, 31),
			},
		},
		{
			date: Date(2020, 6, 1),
			result: map[Period]time.Time{
				Quarterly: Date(2020, 6, 30),
			},
		},
		{
			date: Date(2020, 12, 31),
			result: map[Period]time.Time{
				Quarterly: Date(2020, 12, 31),
			},
		},
	}

	for _, test := range tests {
		for period, result := range test.result {
			if got := EndOf(test.date, period); got != result {
				t.Errorf("EndOf(%v, %v): Got %v, wanted %v", test.date, period, got, result)
			}
		}
	}
}

func TestSeries(t *testing.T) {
	tests := []struct {
		t0     time.Time
		t1     time.Time
		period Period
		result []time.Time
	}{
		{
			t0:     Date(2020, 5, 19),
			t1:     Date(2020, 5, 22),
			period: Daily,
			result: []time.Time{Date(2020, 5, 18), Date(2020, 5, 19), Date(2020, 5, 20), Date(2020, 5, 21), Date(2020, 5, 22)},
		},
		{
			t0:     Date(2020, 1, 1),
			t1:     Date(2020, 1, 31),
			period: Weekly,
			result: []time.Time{Date(2019, 12, 29), Date(2020, 1, 5), Date(2020, 1, 12), Date(2020, 1, 19), Date(2020, 1, 26), Date(2020, 2, 2)},
		},
		{
			t0:     Date(2020, 1, 1),
			t1:     Date(2020, 1, 31),
			period: Monthly,
			result: []time.Time{Date(2019, 12, 31), Date(2020, 1, 31)},
		},
		{
			t0:     Date(2017, 4, 1),
			t1:     Date(2019, 3, 3),
			period: Yearly,
			result: []time.Time{Date(2016, 12, 31), Date(2017, 12, 31), Date(2018, 12, 31), Date(2019, 12, 31)},
		},
	}

	for _, test := range tests {
		if got := Series(test.t0, test.t1, test.period); !Equal(got, test.result) {
			t.Errorf("Series(%v, %v, %v): Got %v, wanted %v", test.t0, test.t1, test.period, got, test.result)
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
