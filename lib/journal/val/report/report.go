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
	"context"
	"time"

	"github.com/sboehler/knut/lib/common/amounts"
	"github.com/sboehler/knut/lib/common/cpr"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/journal/ast"
	"github.com/sboehler/knut/lib/journal/val"
	"github.com/shopspring/decimal"
)

// Report is a balance report for a range of dates.
type Report struct {
	Dates     []time.Time
	Mapping   journal.Mapping
	Positions indexByAccount
}

// Subtree returns the accounts of the minimal dense subtree which
// covers the accounts in this report.
func (rep Report) Subtree() map[*journal.Account]struct{} {
	m := make(map[*journal.Account]struct{})
	for acc := range rep.Positions {
		for p := acc; p != nil; p = p.Parent() {
			m[p] = struct{}{}
		}
	}
	return m
}

// Builder builds a report.
type Builder struct {
	Mapping journal.Mapping

	Result *Report
}

func (rb *Builder) add(rep *Report, b *val.Day) {
	rep.Dates = append(rep.Dates, b.Date)
	if rep.Positions == nil {
		rep.Positions = make(indexByAccount)
	}
	for pos, val := range b.Values {
		if val.IsZero() {
			continue
		}
		if acc := pos.Account.Map(rb.Mapping); acc != nil {
			rep.Positions.Add(acc, pos.Commodity, b.Date, val)
		}
	}
}

// FromStream consumes the stream and produces a report.
func (rb *Builder) FromStream(ctx context.Context, ch <-chan *val.Day) (<-chan *Report, <-chan error) {
	var (
		resCh = make(chan *Report)
		errCh = make(chan error)
	)
	go func() {
		defer close(resCh)
		defer close(errCh)

		res := new(Report)
		for {
			d, ok, err := cpr.Pop(ctx, ch)
			if !ok {
				break
			}
			if err != nil {
				return
			}
			rb.add(res, d)
		}
		cpr.Push(ctx, resCh, res)
	}()
	return resCh, errCh
}

// Push adds values to the report.
func (rb *Builder) Push(ctx context.Context, d ast.Dated) error {
	if rb.Result == nil {
		rb.Result = new(Report)
	}
	if v, ok := d.Elem.(amounts.Amounts); ok {
		rb.Result.Dates = append(rb.Result.Dates, d.Date)
		if rb.Result.Positions == nil {
			rb.Result.Positions = make(indexByAccount)
		}
		for pos, val := range v {
			if val.IsZero() {
				continue
			}
			if acc := pos.Account.Map(rb.Mapping); acc != nil {
				rb.Result.Positions.Add(acc, pos.Commodity, d.Date, val)
			}
		}
	}
	return nil
}

type indexByAccount map[*journal.Account]indexByCommodity

func (iba indexByAccount) Add(acc *journal.Account, com *journal.Commodity, date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	byCommodity, ok := iba[acc]
	if !ok {
		byCommodity = make(indexByCommodity)
		iba[acc] = byCommodity
	}
	byCommodity.Add(com, date, val)
}

type indexByCommodity map[*journal.Commodity]indexByDate

func (ibc indexByCommodity) AddFrom(otherByCommodity indexByCommodity) {
	for c, otherByDate := range otherByCommodity {
		for d, v := range otherByDate {
			ibc.Add(c, d, v)
		}
	}
}

func (ibc indexByCommodity) Normalize() {
	for com, byDate := range ibc {
		if byDate.IsZero() {
			delete(ibc, com)
		}
	}
}

func (ibc indexByCommodity) Sum() map[time.Time]decimal.Decimal {
	res := make(indexByDate)
	for _, byDate := range ibc {
		res.AddFrom(byDate)
	}
	return res
}

func (ibc indexByCommodity) Add(com *journal.Commodity, date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	byDate, ok := ibc[com]
	if !ok {
		byDate = make(indexByDate)
		ibc[com] = byDate
	}
	byDate.Add(date, val)
}

type indexByDate map[time.Time]decimal.Decimal

func (ibd indexByDate) Add(date time.Time, val decimal.Decimal) {
	if val.IsZero() {
		return
	}
	ibd[date] = ibd[date].Add(val)
}

func (ibd indexByDate) AddFrom(obd indexByDate) {
	for d, val := range obd {
		ibd.Add(d, val)
	}
}

func (ibd indexByDate) IsZero() bool {
	for _, val := range ibd {
		if !val.IsZero() {
			return false
		}
	}
	return true
}
