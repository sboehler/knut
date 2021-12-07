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

package balance

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/prices"
)

// SetDate sets the date on the balance.
func SetDate(ctx context.Context, bsCh chan *Balance, dayCh chan *ledger.Day) <-chan *Balance {
	resCh := make(chan *Balance)
	go func() {
		defer close(resCh)
		for day := range dayCh {
			bal := <-bsCh
			bal.Date = day.Date
			select {
			case resCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return resCh
}

// SnapshotConfig configures balance snapshotting.
type SnapshotConfig struct {
	Ledger   *ledger.Ledger
	From, To *time.Time
	Last     int
	Diff     bool
	Period   date.Period
}

// Snapshot snapshots the balance.
func Snapshot(ctx context.Context, cfg SnapshotConfig, bs <-chan *Balance, dayCh <-chan *ledger.Day) (<-chan *Balance, <-chan *Balance) {
	dates := cfg.Ledger.Dates(cfg.From, cfg.To, cfg.Period)
	offset := 0
	if cfg.Diff {
		offset = 1
	}
	if cfg.Last > 0 && cfg.Last < len(dates)-offset {
		dates = dates[len(dates)-cfg.Last-offset:]
	}
	var (
		snapshotDates []time.Time
		i             int
	)
	for _, day := range cfg.Ledger.Days {
		for ; i < len(dates) && day.Date.After(dates[i]); i++ {
			snapshotDates = append(snapshotDates, day.Date)
		}
	}
	maxDate, ok := cfg.Ledger.MaxDate()
	if ok {
		for ; i < len(dates); i++ {
			snapshotDates = append(snapshotDates, maxDate)
		}
	}
	nextCh := make(chan *Balance)
	snapshotCh := make(chan *Balance)

	go func() {
		defer close(nextCh)
		var (
			previous *Balance
			index    int
		)
		for day := range dayCh {
			bal := <-bs
			for ; index < len(snapshotDates) && snapshotDates[index] == day.Date; index++ {
				snapshot := bal.Snapshot()
				snapshot.Date = dates[index]
				if cfg.Diff {
					if previous != nil {
						diff := snapshot.Snapshot()
						diff.Minus(previous)
						snapshotCh <- diff
					}
					previous = snapshot
				} else {
					snapshotCh <- snapshot
				}
			}
			select {
			case nextCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nextCh, snapshotCh
}

// UpdatePrices updates the prices.
func UpdatePrices(ctx context.Context, val *ledger.Commodity, bs <-chan *Balance, dayCh <-chan *ledger.Day) <-chan *Balance {
	ps := make(prices.Prices)
	if val == nil {
		return bs
	}
	buf := make(chan prices.NormalizedPrices, 50)
	go func() {
		defer close(buf)
		for day := range dayCh {
			for _, p := range day.Prices {
				ps.Insert(p)
			}
			select {
			case buf <- ps.Normalize(val):
			case <-ctx.Done():
				return
			}
		}
	}()
	nextCh := make(chan *Balance)
	go func() {
		defer close(nextCh)
		for bal := range bs {
			bal.NormalizedPrices = <-buf
			select {
			case nextCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nextCh
}
