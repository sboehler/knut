package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/server/model"
	"github.com/shopspring/decimal"
)

func TestCreateTransaction(t *testing.T) {
	var (
		date     = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		bookings = []model.Booking{
			{
				Amount:          decimal.RequireFromString("4.23"),
				CommodityID:     1,
				CreditAccountID: 1,
				DebitAccountID:  2,
			},
		}
		trx = model.Transaction{
			Date:        date,
			Description: "desc",
			Bookings:    bookings,
		}
		want = model.Transaction{
			ID:          1,
			Date:        date,
			Description: "desc",
			Bookings:    bookings,
		}
	)
	populateCommodities(ctx, t, db, []string{"AAA"})
	populateAccounts(ctx, t, db, []model.Account{{Name: "Foo"}, {Name: "Bar"}})

	got, err := CreateTransaction(ctx, db, trx)

	if err != nil {
		t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CreateTransaction() mismatch (-want +got):\n%s", diff)
	}
}

func TestListTransaction(t *testing.T) {
	var (
		date1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		date2 = time.Date(2021, time.May, 15, 0, 0, 0, 0, time.UTC)
		date3 = time.Date(2021, time.May, 16, 0, 0, 0, 0, time.UTC)
		ctx   = context.Background()
		db    = createAndMigrateInMemoryDB(ctx, t)
		trx   = []model.Transaction{
			{
				ID:          1,
				Date:        date1,
				Description: "desc1",
				Bookings: []model.Booking{
					{
						ID:              1,
						Amount:          decimal.RequireFromString("4.23"),
						CommodityID:     1,
						CreditAccountID: 1,
						DebitAccountID:  2,
					},
				},
			},
			{
				ID:          2,
				Date:        date2,
				Description: "desc2",
				Bookings: []model.Booking{
					{
						ID:              2,
						Amount:          decimal.RequireFromString("3.23"),
						CommodityID:     1,
						CreditAccountID: 2,
						DebitAccountID:  1,
					},
				},
			},
			{
				ID:          3,
				Date:        date3,
				Description: "desc3",
				Bookings: []model.Booking{
					{
						ID:              3,
						Amount:          decimal.RequireFromString("3.23"),
						CommodityID:     1,
						CreditAccountID: 2,
						DebitAccountID:  1,
					},
				},
			},
		}
	)
	populateCommodities(ctx, t, db, []string{"AAA"})
	populateAccounts(ctx, t, db, []model.Account{{Name: "Foo"}, {Name: "Bar"}})
	populateTransactions(ctx, t, db, trx)

	got, err := ListTransactions(ctx, db)

	if err != nil {
		t.Fatalf("ListTransactions() returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(trx, got); diff != "" {
		t.Errorf("ListTransactions() mismatch (-want +got):\n%s", diff)
	}
}

func populateTransactions(ctx context.Context, t *testing.T, db db, ts []model.Transaction) []model.Transaction {
	t.Helper()
	var res []model.Transaction
	for _, trx := range ts {
		trx, err := CreateTransaction(ctx, db, trx)
		if err != nil {
			t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
		}
		res = append(res, trx)
	}
	return res
}
