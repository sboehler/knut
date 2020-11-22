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

package ledger

import (
	"io"
	"time"

	"github.com/sboehler/knut/lib/model"
)

// Step groups all commands for a given date.
type Step struct {
	Date         time.Time
	Prices       []*model.Price
	Assertions   []*model.Assertion
	Openings     []*model.Open
	Transactions []*model.Transaction
	Closings     []*model.Close
}

// Ledger is a ledger.
type Ledger []*Step

// MinDate returns the minimum date for this ledger, as the first
// date on which an account is opened (ignoring prices, for example).
func (l Ledger) MinDate() (time.Time, bool) {
	for _, s := range l {
		if len(s.Openings) > 0 {
			return s.Date, true
		}
	}
	return time.Time{}, false
}

// MaxDate returns the maximum date for the given ledger.
func (l Ledger) MaxDate() (time.Time, bool) {
	if len(l) == 0 {
		return time.Time{}, false
	}
	return l[len(l)-1].Date, true
}

// WriteTo pretty-prints the ledger to the given writer.
func (l Ledger) WriteTo(w io.Writer) (int64, error) {
	var n int64
	for _, step := range l {
		for _, p := range step.Prices {
			if err := write(w, p, &n); err != nil {
				return n, err
			}
		}
		for _, o := range step.Openings {
			if err := write(w, o, &n); err != nil {
				return n, err
			}
		}
		for _, t := range step.Transactions {
			if err := write(w, t, &n); err != nil {
				return n, err
			}
		}
		for _, a := range step.Assertions {
			if err := write(w, a, &n); err != nil {
				return n, err
			}
		}
		for _, c := range step.Closings {
			if err := write(w, c, &n); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// Write writes the given WriterTo to the Writer, followed
// by a blank line.
func write(w io.Writer, wr io.WriterTo, count *int64) error {
	c, err := wr.WriteTo(w)
	*count += c
	if err != nil {
		return err
	}
	d, err := w.Write([]byte{'\n'})
	*count += int64(d)
	if err != nil {
		return err
	}
	return nil
}
