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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestStartOf(t *testing.T) {
	tests := []struct {
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
	tests := []struct {
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

func TestPeriodDates(t *testing.T) {
	tests := []struct {
		period   Period
		interval Interval
		result   []time.Time
	}{
		{
			period:   Period{Start: Date(2020, 5, 19), End: Date(2020, 5, 22)},
			interval: Once,
			result:   []time.Time{Date(2020, 5, 22)},
		},
		{
			period:   Period{Start: Date(2020, 5, 19), End: Date(2020, 5, 22)},
			interval: Daily,
			result: []time.Time{
				Date(2020, 5, 19),
				Date(2020, 5, 20),
				Date(2020, 5, 21),
				Date(2020, 5, 22),
			},
		},
		{
			period:   Period{Start: Date(2020, 1, 1), End: Date(2020, 1, 31)},
			interval: Weekly,
			result: []time.Time{
				Date(2020, 1, 5),
				Date(2020, 1, 12),
				Date(2020, 1, 19),
				Date(2020, 1, 26),
				Date(2020, 1, 31),
			},
		},
		{
			period:   Period{Start: Date(2019, 12, 31), End: Date(2020, 1, 31)},
			interval: Monthly,
			result: []time.Time{
				Date(2019, 12, 31),
				Date(2020, 1, 31),
			},
		},
		{
			period:   Period{Start: Date(2020, 1, 1), End: Date(2020, 1, 31)},
			interval: Monthly,
			result:   []time.Time{Date(2020, 1, 31)},
		},
		{
			period:   Period{Start: Date(2017, 4, 1), End: Date(2019, 3, 3)},
			interval: Yearly,
			result: []time.Time{
				Date(2017, 12, 31),
				Date(2018, 12, 31),
				Date(2019, 3, 3),
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {

			got := test.period.Dates(test.interval, 0)

			if diff := cmp.Diff(test.result, got); diff != "" {
				t.Fatalf("Periods(%v, %v): unexpected diff (+got/-want):\n%s", test.period, test.interval, diff)
			}
		})
	}
}
