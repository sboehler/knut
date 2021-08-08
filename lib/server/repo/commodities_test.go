package repo

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/server/model"
)

func TestCreateCommodity(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Commodities: []model.Commodity{{Name: "CHF"}},
		})
		tests = []struct {
			desc    string
			name    string
			wantErr error
		}{
			{
				desc: "commodity does not exist",
				name: "USD",
			},
			{
				desc:    "existing commodity",
				name:    "CHF",
				wantErr: cmpopts.AnyError,
			},
		}
	)

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := beginTransaction(ctx, t, db)
			defer db.Rollback()

			c, err := CreateCommodity(ctx, db, test.name)
			if !cmp.Equal(test.wantErr, err, cmpopts.EquateErrors()) {
				t.Fatalf("CreateCommodity returned error %v, expected %v", err, test.wantErr)
				return
			}
			if test.wantErr != nil {
				return
			}
			var (
				want = Scenario{Commodities: append(scenario.Commodities, c)}
				got  = Load(ctx, t, db)
			)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("CreateCommodityPrice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestListCommodity(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Commodities: []model.Commodity{
				{Name: "CHF"}, {Name: "EUR"}, {Name: "USD"},
			},
		})
	)

	got, err := ListCommodities(ctx, db)

	if err != nil {
		t.Errorf("ListCommodities() returned unexpected error: %v", err)
	}
	sort.Slice(got, func(i, j int) bool {
		return got[i].Less(got[j])
	})
	if diff := cmp.Diff(scenario.Commodities, got); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}
}

func TestListCommodityConcurrently(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Scenario{
			Commodities: []model.Commodity{
				{Name: "CHF"}, {Name: "EUR"}, {Name: "USD"},
			},
		}
	)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error on BeginTx(): %v", err)
	}
	scenario = Save(ctx, t, tx, scenario)
	if err := tx.Commit(); err != nil {
		t.Fatalf("unexpected error on commit: %v", err)
	}
	var mu sync.RWMutex
	for i := 0; i < 1000; i++ {
		i := i
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			t.Parallel()

			if i%10 == 0 {
				mu.Lock()
				defer mu.Unlock()
				_, err := CreateCommodity(ctx, db, fmt.Sprintf("C%d", i))
				if err != nil {
					t.Fatalf("CreateCommodities() returned unexpected error: %v", err)
				}

			} else {
				mu.RLock()
				defer mu.RUnlock()

				_, err := ListCommodities(ctx, db)

				if err != nil {
					t.Fatalf("ListCommodities() returned unexpected error: %v", err)
				}
			}
		})
	}
}
