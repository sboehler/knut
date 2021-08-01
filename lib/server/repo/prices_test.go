package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/server/model"
)

func TestInsertPrice(t *testing.T) {
	type tipTest struct {
		price model.Price
		want  Scenario
	}

	var (
		date1, date2 = date.Date(2021, 5, 1), date.Date(2021, 5, 2)
		ctx          = context.Background()
		db           = createAndMigrateInMemoryDB(ctx, t)
		s            = Save(ctx, t, db, Scenario{
			Commodities: []model.Commodity{{Name: "CHF"}, {Name: "USD"}, {Name: "EUR"}},
			Prices: []model.Price{
				{Date: date1, CommodityID: 0, TargetCommodityID: 1, Price: 10},
			},
		})

		c0, c1, c2 = s.Commodities[0], s.Commodities[1], s.Commodities[2]

		tests = []struct {
			desc   string
			insert model.Price
			want   Scenario
		}{
			{
				desc:   "modify existing price with new value",
				insert: model.Price{Date: date1, CommodityID: c0.ID, TargetCommodityID: c1.ID, Price: 20},
				want: Scenario{
					Commodities: s.Commodities,
					Prices: []model.Price{
						{Date: date1, CommodityID: c0.ID, TargetCommodityID: c1.ID, Price: 20},
					},
				},
			},
			{
				desc:   "same commodities and value, new date",
				insert: model.Price{Date: date2, CommodityID: c0.ID, TargetCommodityID: c1.ID, Price: 20},
				want: Scenario{
					Commodities: s.Commodities,
					Prices: append(
						s.Prices,
						model.Price{Date: date2, CommodityID: c0.ID, TargetCommodityID: c1.ID, Price: 20}),
				},
			},
			{
				desc:   "price with new commodity",
				insert: model.Price{Date: date2, CommodityID: c0.ID, TargetCommodityID: c2.ID, Price: 200},
				want: Scenario{
					Commodities: s.Commodities,
					Prices: append(
						s.Prices,
						model.Price{Date: date2, CommodityID: c0.ID, TargetCommodityID: c2.ID, Price: 200}),
				},
			},
		}
	)

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var tx = beginTransaction(ctx, t, db)
			defer tx.Rollback()

			_, err := InsertPrice(ctx, tx, test.insert)

			if err != nil {
				t.Fatalf("InsertPrice() returned unexpected error: %v", err)
			}
			var got = Load(ctx, t, tx)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("InsertPrice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func beginTransaction(ctx context.Context, t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("error beginning transaction %v", err)
	}
	return tx

}

func TestListPrices(t *testing.T) {
	var (
		date1, date2, date3 = date.Date(2021, time.May, 14), date.Date(2021, time.May, 15), date.Date(2021, time.May, 16)
		ctx                 = context.Background()
		db                  = createAndMigrateInMemoryDB(ctx, t)
		scenario            = Save(ctx, t, db, Scenario{
			Commodities: []model.Commodity{{Name: "CHF"}, {Name: "USD"}},
			Prices: []model.Price{
				{Date: date1, CommodityID: 0, TargetCommodityID: 1, Price: 10},
				{Date: date2, CommodityID: 0, TargetCommodityID: 1, Price: 11},
				{Date: date3, CommodityID: 0, TargetCommodityID: 1, Price: 12},
				{Date: date1, CommodityID: 1, TargetCommodityID: 0, Price: 13},
			},
		})
	)

	got, err := ListPrices(ctx, db)

	if err != nil {
		t.Fatalf("ListPrices returned unexpected error: %v", err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Less(got[j]) })
	if diff := cmp.Diff(scenario.Prices, got); diff != "" {
		t.Errorf("ListPrices() mismatch (-want +got):\n%s", diff)
	}
}

func TestDeletePrices(t *testing.T) {
	var (
		date1    = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		date2    = date1.AddDate(0, 0, 1)
		date3    = date2.AddDate(0, 0, 1)
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Commodities: []model.Commodity{{Name: "USD"}, {Name: "CHF"}},
			Prices: []model.Price{
				{Date: date1, CommodityID: 0, TargetCommodityID: 1, Price: 10},
				{Date: date2, CommodityID: 0, TargetCommodityID: 1, Price: 11},
				{Date: date3, CommodityID: 1, TargetCommodityID: 1, Price: 12},
				{Date: date3, CommodityID: 1, TargetCommodityID: 0, Price: 13},
			}})
	)

	for i, pr := range scenario.Prices {
		t.Run(fmt.Sprintf("delete price no %d", i), func(t *testing.T) {
			var db = beginTransaction(ctx, t, db)
			defer db.Rollback()

			err := DeletePrice(ctx, db, pr.Date, pr.CommodityID, pr.TargetCommodityID)

			if err != nil {
				t.Fatalf("DeletePrice() returned unexpected error: %v", err)
			}
			var want = Scenario{Commodities: scenario.Commodities}
			for j, p2 := range scenario.Prices {
				if i != j {
					want.Prices = append(want.Prices, p2)
				}
			}
			got := Load(ctx, t, db)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("DeletePrice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func populatePrices(ctx context.Context, t *testing.T, db db, prices []model.Price) {
	t.Helper()
	for _, price := range prices {
		if _, err := InsertPrice(ctx, db, price); err != nil {
			t.Fatalf("InsertPrice() returned unexpected error: %v", err)
		}
	}
}
