package repo

import (
	"context"
	"sort"
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

	tx := beginTransaction(ctx, t, db)
	defer tx.Rollback()

	got, err := ListCommodities(ctx, tx)

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
