package repo

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/server/model"
	"github.com/shopspring/decimal"
)

func createTransactionScenario(ctx context.Context, t *testing.T, db db) Scenario {
	return Save(ctx, t, db, Scenario{
		Commodities: []model.Commodity{{Name: "AAA"}, {Name: "BBB"}, {Name: "CCC"}},
		Accounts:    []model.Account{{Name: "Foo"}, {Name: "Bar"}, {Name: "Baz"}},
		Transactions: []model.Transaction{
			{
				Date: date.Date(2021, time.May, 14), Description: "desc1", Bookings: []model.Booking{
					{Amount: decimal.RequireFromString("4.23"), CommodityID: 0, CreditAccountID: 0, DebitAccountID: 1},
					{Amount: decimal.RequireFromString("3.23"), CommodityID: 1, CreditAccountID: 1, DebitAccountID: 0},
				},
			},
			{
				Date: date.Date(2021, time.May, 15), Description: "desc2", Bookings: []model.Booking{
					{Amount: decimal.RequireFromString("8.90"), CommodityID: 0, CreditAccountID: 0, DebitAccountID: 1},
					{Amount: decimal.RequireFromString("20.3"), CommodityID: 1, CreditAccountID: 1, DebitAccountID: 2},
				},
			},
			{
				Date: date.Date(2021, time.May, 16), Description: "desc3", Bookings: []model.Booking{
					{Amount: decimal.RequireFromString("3.23"), CommodityID: 2, CreditAccountID: 0, DebitAccountID: 1},
				},
			},
		},
	})
}

func TestCreateTransaction(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = createTransactionScenario(ctx, t, db)
		tests    = []struct {
			desc string
			trx  model.Transaction
		}{
			{
				desc: "new transaction",
				trx: model.Transaction{
					Date:        date.Date(2021, time.May, 31),
					Description: "new transaction",
					Bookings: []model.Booking{
						{

							Amount:          decimal.NewFromInt(200),
							CommodityID:     scenario.Commodities[0].ID,
							CreditAccountID: scenario.Accounts[0].ID,
							DebitAccountID:  scenario.Accounts[1].ID,
						},
					},
				},
			},
		}
	)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := beginTransaction(ctx, t, db)
			defer db.Rollback()

			created, err := CreateTransaction(ctx, db, test.trx)

			if err != nil {
				t.Fatalf("CreateTransaction() returned unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.trx, created, cmpopts.IgnoreFields(created, "ID")); diff != "" {
				t.Errorf("CreateTransaction() mismatch (-want +got):\n%s", diff)
			}
			got := Load(ctx, t, db)
			want := Scenario{
				Commodities:  scenario.Commodities,
				Accounts:     scenario.Accounts,
				Transactions: append(scenario.Transactions, created),
			}
			sort.Slice(want.Transactions, func(i, j int) bool { return want.Transactions[i].Less(want.Transactions[j]) })
			for _, t := range want.Transactions {
				sort.Slice(t.Bookings, func(i, j int) bool { return t.Bookings[i].Less(t.Bookings[j]) })
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("CreateTransaction() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestListTransaction(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = createTransactionScenario(ctx, t, db)
	)

	got, err := ListTransactions(ctx, db)

	if err != nil {
		t.Fatalf("ListTransactions() returned unexpected error: %v", err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Less(got[j]) })
	for _, t := range got {
		sort.Slice(t.Bookings, func(i, j int) bool { return t.Bookings[i].Less(t.Bookings[j]) })
	}
	if diff := cmp.Diff(scenario.Transactions, got); diff != "" {
		t.Errorf("ListTransactions() mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateTransaction(t *testing.T) {
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = createTransactionScenario(ctx, t, db)

		update = model.Transaction{
			ID: scenario.Transactions[0].ID, Date: date.Date(2021, time.June, 1), Description: "desc4",
			Bookings: []model.Booking{
				{
					Amount:          decimal.RequireFromString("15.23"),
					CommodityID:     scenario.Commodities[0].ID,
					CreditAccountID: scenario.Accounts[2].ID,
					DebitAccountID:  scenario.Accounts[0].ID,
				},
			},
		}
	)

	updated, err := UpdateTransaction(ctx, db, update)

	if err != nil {
		t.Fatalf("UpdateTransaction() returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(update, updated); diff != "" {
		t.Errorf("UpdateTransaction() mismatch (-want +got):\n%s", diff)
	}
	var (
		got  = Load(ctx, t, db)
		want = Scenario{
			Commodities:  scenario.Commodities,
			Accounts:     scenario.Accounts,
			Transactions: append([]model.Transaction{updated}, scenario.Transactions[1:]...)}
	)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("UpdateTransactions() mismatch (-want +got):\n%s", diff)
	}
}
