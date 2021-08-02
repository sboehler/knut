package repo

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/sboehler/knut/lib/server/database"
	"github.com/sboehler/knut/lib/server/model"
)

func createAndMigrateInMemoryDB(ctx context.Context, t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("error creating in-memory database: %v", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("error when enabling foreign keys: %v", err)
	}
	return db
}

type Scenario struct {
	Commodities  []model.Commodity
	Accounts     []model.Account
	Prices       []model.Price
	Transactions []model.Transaction
}

// Save sets up a new scenario.
func Save(ctx context.Context, t *testing.T, db db, s Scenario) Scenario {
	t.Helper()
	var sp Scenario
	for _, c := range s.Commodities {
		cp, err := CreateCommodity(ctx, db, c.Name)
		if err != nil {
			t.Fatalf("failed to insert commodity %v with error %v", c, err)
		}
		sp.Commodities = append(sp.Commodities, cp)
	}
	for _, a := range s.Accounts {
		ap, err := CreateAccount(ctx, db, a.Name, a.OpenDate, a.CloseDate)
		if err != nil {
			t.Fatalf("failed to insert account %v with error %v", a, err)
		}
		sp.Accounts = append(sp.Accounts, ap)
	}
	for _, p := range s.Prices {
		p.CommodityID = sp.Commodities[p.CommodityID].ID
		p.TargetCommodityID = sp.Commodities[p.TargetCommodityID].ID
		pp, err := InsertPrice(ctx, db, p)
		if err != nil {
			t.Fatalf("failed to insert price %v with error %v", p, err)
		}
		sp.Prices = append(sp.Prices, pp)
	}
	for _, trx := range s.Transactions {
		var bp []model.Booking
		for _, b := range trx.Bookings {
			b.CommodityID = sp.Commodities[b.CommodityID].ID
			b.CreditAccountID = sp.Accounts[b.CreditAccountID].ID
			b.DebitAccountID = sp.Accounts[b.DebitAccountID].ID
			bp = append(bp, b)
		}
		trx.Bookings = bp
		trxp, err := CreateTransaction(ctx, db, trx)
		if err != nil {
			t.Fatalf("failed to insert transaction %v with error %v", trx, err)
		}
		sp.Transactions = append(sp.Transactions, trxp)
	}
	sp.Normalize()
	return sp
}

func Load(ctx context.Context, t *testing.T, db db) Scenario {
	t.Helper()
	var (
		res Scenario
		err error
	)
	if res.Commodities, err = ListCommodities(ctx, db); err != nil {
		t.Fatalf("ListCommodities returned an unexpected error %v", err)
	}
	if res.Accounts, err = ListAccounts(ctx, db); err != nil {
		t.Fatalf("ListAccounts returned an unexpected error %v", err)
	}
	if res.Prices, err = ListPrices(ctx, db); err != nil {
		t.Fatalf("ListPrices returned an unexpected error %v", err)
	}
	if res.Transactions, err = ListTransactions(ctx, db); err != nil {
		t.Fatalf("ListTransactions returned an unexpected error %v", err)
	}
	return res.Normalize()
}

func (s Scenario) Normalize() Scenario {
	sort.Slice(s.Commodities, func(i, j int) bool {
		return s.Commodities[i].Less(s.Commodities[j])
	})
	sort.Slice(s.Accounts, func(i, j int) bool {
		return s.Accounts[i].ID < s.Accounts[j].ID
	})
	sort.Slice(s.Prices, func(i, j int) bool {
		return s.Prices[i].Less(s.Prices[j])
	})
	for _, t := range s.Transactions {
		sort.Slice(t.Bookings, func(i, j int) bool {
			return t.Bookings[i].Less(t.Bookings[j])
		})
	}
	sort.Slice(s.Transactions, func(i, j int) bool {
		return s.Transactions[i].Less(s.Transactions[j])
	})
	return s
}

func (s Scenario) DeepCopy() Scenario {
	var copy Scenario
	for _, c := range s.Commodities {
		copy.Commodities = append(copy.Commodities, c)
	}
	for _, a := range s.Accounts {
		copy.Accounts = append(copy.Accounts, a)
	}
	for _, p := range s.Prices {
		copy.Prices = append(copy.Prices, p)
	}
	for _, trx := range s.Transactions {
		var bp []model.Booking
		for _, b := range trx.Bookings {
			b.CommodityID = copy.Commodities[b.CommodityID].ID
			bp = append(bp, b)
		}
		trx.Bookings = bp
		copy.Transactions = append(copy.Transactions, trx)
	}
	return copy
}
