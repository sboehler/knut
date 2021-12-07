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
func SetDate(ctx context.Context, l ledger.Ledger, bsCh chan *Balance) <-chan *Balance {
	nextCh := make(chan *Balance)
	go func() {
		defer close(nextCh)
		var index int
		for bal := range bsCh {
			bal.Date = l.Days[index].Date
			index++
			select {
			case nextCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nextCh
}

// SnapshotConfig configures balance snapshotting.
type SnapshotConfig struct {
	From, To *time.Time
	Last     int
	Diff     bool
	Period   date.Period
}

// Snapshot snapshots the balance.
func Snapshot(ctx context.Context, cfg SnapshotConfig, l ledger.Ledger, bs <-chan *Balance) <-chan *Balance {
	dates := l.Dates(cfg.From, cfg.To, cfg.Period)
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
	for _, day := range l.Days {
		for ; i < len(dates) && day.Date.After(dates[i]); i++ {
			snapshotDates = append(snapshotDates, day.Date)
		}
	}
	maxDate, ok := l.MaxDate()
	if ok {
		for ; i < len(dates); i++ {
			snapshotDates = append(snapshotDates, maxDate)
		}
	}
	snapshotCh := make(chan *Balance)

	go func() {
		defer close(snapshotCh)
		var (
			previous *Balance
			index    int
		)
		for bal := range bs {
			for ; index < len(snapshotDates) && snapshotDates[index] == bal.Date; index++ {
				snapshot := bal.Snapshot()
				snapshot.Date = dates[index]
				if cfg.Diff {
					if previous != nil {
						diff := snapshot.Snapshot()
						diff.Minus(previous)
						select {
						case snapshotCh <- diff:
						case <-ctx.Done():
						}
					}
					previous = snapshot
				} else {
					select {
					case snapshotCh <- snapshot:
					case <-ctx.Done():
					}
				}
			}
		}
	}()
	return snapshotCh
}

// UpdatePrices updates the prices.
func UpdatePrices(ctx context.Context, l ledger.Ledger, val *ledger.Commodity, bs <-chan *Balance) <-chan *Balance {
	ps := make(prices.Prices)
	if val == nil {
		return bs
	}
	buf := make(chan prices.NormalizedPrices, 50)
	go func() {
		defer close(buf)
		for _, day := range l.Days {
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
