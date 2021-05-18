package repo

import (
	"context"
	"database/sql"
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
	return db
}

func populateCommodities(ctx context.Context, t *testing.T, db *sql.DB, names []string) []model.Commodity {
	t.Helper()
	var res []model.Commodity
	for _, name := range names {
		c, err := CreateCommodity(ctx, db, name)
		if err != nil {
			t.Fatalf("Create(ctx, %s) returned unexpected error: %v", name, err)
		}
		res = append(res, c)
	}
	return res
}
