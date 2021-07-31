package repo

import (
	"context"
	"fmt"
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
		date1    = date.Date(2021, 5, 1)
		date2    = date.Date(2021, 5, 2)
		ctx      = context.Background()
		scenario = Scenario{
			Commodities: []model.Commodity{
				{Name: "CHF"},
				{Name: "USD"},
			},
			Prices: []model.Price{
				{Date: date1, CommodityID: 0, TargetCommodityID: 1, Price: 10},
			},
		}

		tests = []func(Scenario) tipTest{
			func(s Scenario) tipTest {
				return tipTest{
					price: model.Price{
						Date:              date1,
						CommodityID:       s.Commodities[0].ID,
						TargetCommodityID: s.Commodities[1].ID,
						Price:             20,
					},
					want: Scenario{
						Commodities: s.Commodities,
						Prices: []model.Price{
							{
								Date:              date1,
								CommodityID:       s.Commodities[0].ID,
								TargetCommodityID: s.Commodities[1].ID,
								Price:             20,
							},
						},
					},
				}
			},
			func(s Scenario) tipTest {
				return tipTest{
					price: model.Price{Date: date2,
						CommodityID:       s.Commodities[0].ID,
						TargetCommodityID: s.Commodities[1].ID,
						Price:             20,
					},
					want: Scenario{
						Commodities: s.Commodities,
						Prices: append(s.Prices, model.Price{
							Date:              date2,
							CommodityID:       s.Commodities[0].ID,
							TargetCommodityID: s.Commodities[1].ID,
							Price:             20,
						}),
					},
				}
			},
		}
	)

	for _, test := range tests {
		var (
			db   = createAndMigrateInMemoryDB(ctx, t)
			s    = Save(ctx, t, db, scenario.DeepCopy())
			test = test(s)
		)

		_, err := InsertPrice(ctx, db, test.price)

		fmt.Println(s, test, err)

		if err != nil {
			t.Fatalf("InsertPrice() returned unexpected error: %v", err)
		}
		var got = Load(ctx, t, db)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("InsertPrice() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestListPrices(t *testing.T) {
	var (
		date1  = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		date2  = date1.AddDate(0, 0, 1)
		date3  = date2.AddDate(0, 0, 1)
		ctx    = context.Background()
		db     = createAndMigrateInMemoryDB(ctx, t)
		prices = []model.Price{
			{
				Date:              date1,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             10,
			},
			{
				Date:              date2,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             11,
			},
			{
				Date:              date3,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             12,
			},
			{
				Date:              date3,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             13,
			},
		}
		want = []model.Price{
			prices[0],
			prices[1],
			prices[3],
		}
	)
	_ = populateCommodities(ctx, t, db, []string{"AAA"})
	populatePrices(ctx, t, db, prices)

	got, err := ListPrices(ctx, db)

	if err != nil {
		t.Fatalf("ListPrices returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListPrices() mismatch (-want +got):\n%s", diff)
	}
}

func TestDeletePrices(t *testing.T) {
	var (
		date1  = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		date2  = date1.AddDate(0, 0, 1)
		date3  = date2.AddDate(0, 0, 1)
		ctx    = context.Background()
		db     = createAndMigrateInMemoryDB(ctx, t)
		prices = []model.Price{
			{
				Date:              date1,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             10,
			},
			{
				Date:              date2,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             11,
			},
			{
				Date:              date3,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             12,
			},
			{
				Date:              date3,
				CommodityID:       1,
				TargetCommodityID: 1,
				Price:             13,
			},
		}
		want = []model.Price{
			prices[1],
			prices[3],
		}
	)
	_ = populateCommodities(ctx, t, db, []string{"AAA"})
	populatePrices(ctx, t, db, prices)

	err := DeletePrice(ctx, db, prices[0].Date, prices[0].CommodityID, prices[0].TargetCommodityID)

	if err != nil {
		t.Fatalf("DeletePrice() returned unexpected error: %v", err)
	}
	got := listPrices(ctx, t, db)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DeletePrice() mismatch (-want +got):\n%s", diff)
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

func listPrices(ctx context.Context, t *testing.T, db db) []model.Price {
	t.Helper()
	res, err := ListPrices(ctx, db)
	if err != nil {
		t.Fatalf("ListPrices() returned unexpected error: %v", err)
	}
	return res
}
