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

// PreStage sets the date on the balance.
func PreStage(ctx context.Context, l ledger.Ledger, bsCh <-chan *Balance) (<-chan *Balance, <-chan error) {
	nextCh := make(chan *Balance)
	errCh := make(chan error)
	go func() {
		defer close(nextCh)
		defer close(errCh)
		var index int
		for bal := range bsCh {
			day := l.Days[index]
			bal.Date = day.Date

			ps := []ledger.Processor{
				AccountOpener{Balance: bal},
				TransactionBooker{Balance: bal},
				ValueBooker{Balance: bal},
				Asserter{Balance: bal},
			}
			for _, p := range ps {
				if err := p.Process(day); err != nil {
					errCh <- err
					return
				}
			}
			index++
			select {
			case nextCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nextCh, errCh
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

// PostStage sets the date on the balance.
func PostStage(ctx context.Context, l ledger.Ledger, bsCh <-chan *Balance) (<-chan *Balance, <-chan error) {
	nextCh := make(chan *Balance)
	errCh := make(chan error)
	go func() {
		defer close(nextCh)
		defer close(errCh)
		var index int
		for bal := range bsCh {
			day := l.Days[index]
			bal.Date = day.Date

			ps := []ledger.Processor{
				TransactionValuator{Balance: bal},
				ValuationTransactionComputer{Balance: bal},
				AccountCloser{Balance: bal},
			}
			for _, p := range ps {
				if err := p.Process(day); err != nil {
					errCh <- err
					return
				}
			}
			index++
			select {
			case nextCh <- bal:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nextCh, errCh
}

// SnapshotConfig configures balance snapshotting.
type SnapshotConfig struct {
	From, To *time.Time
	Last     int
	Diff     bool
	Period   date.Period
}

// Snapshot snapshots the balance.
func Snapshot(ctx context.Context, cfg SnapshotConfig, l ledger.Ledger, bs <-chan *Balance) (<-chan *Balance, <-chan *Balance) {
	dates := l.Dates(cfg.From, cfg.To, cfg.Period)
	if cfg.Last > 0 {
		last := cfg.Last
		if cfg.Diff {
			last++
		}
		if len(dates) > cfg.Last {
			dates = dates[len(dates)-last:]
		}
	}
	var (
		snapshotDates = l.ActualDates(dates)
		snapshotCh    = make(chan *Balance)
		nextCh        = make(chan *Balance)
	)
	go func() {
		defer close(snapshotCh)
		defer close(nextCh)
		var (
			previous *Balance
			index    int
		)
		// Produce empty balances for dates before the ledger.
		for ; index < len(snapshotDates) && snapshotDates[index].IsZero(); index++ {
			bal := New(l.Context, nil)
			bal.Date = dates[index]
			select {
			case snapshotCh <- New(l.Context, nil):
			case <-ctx.Done():
				return
			}
		}
		// Produce snapshots for dates during the ledger.
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
							return
						}
					}
					previous = snapshot
				} else {
					select {
					case snapshotCh <- snapshot:
					case <-ctx.Done():
						return
					}
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
