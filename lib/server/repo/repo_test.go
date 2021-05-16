package repo

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/server/database"
	"github.com/sboehler/knut/lib/server/model"
)

func TestCreateCommodity(t *testing.T) {
	var (
		ctx     = context.Background()
		db   db = createAndMigrateInMemoryDB(ctx, t)
		name    = "CHF"
		err  error
	)

	c, err := CreateCommodity(ctx, db, name)

	if err != nil {
		t.Errorf("Create(ctx, %s) returned error %v, expected none", name, err)
	}
	if c.Name != "CHF" {
		t.Errorf("Create(ctx, %q) returned commodity with name %q, want %q", name, c.Name, name)
	}
	if c.ID <= 0 {
		t.Errorf("Create(ctx, %s) returned commodity with ID %v, want a positive number", name, c.ID)
	}
}

func TestCreateDuplicateCommodity(t *testing.T) {
	var (
		ctx  = context.Background()
		db   = createAndMigrateInMemoryDB(ctx, t)
		name = "AAA"
	)
	_ = populateCommodities(ctx, t, db, []string{name})

	_, err := CreateCommodity(ctx, db, name)

	if err == nil {
		t.Errorf("Create(ctx, %s) returned no error, but expected one", name)
	}
}

func TestListCommodity(t *testing.T) {
	var (
		ctx   = context.Background()
		db    = createAndMigrateInMemoryDB(ctx, t)
		names = []string{"CCC", "BBB", "AAA"}
		want  = populateCommodities(ctx, t, db, names)
	)
	sort.Slice(want, func(i, j int) bool {
		return want[i].Name < want[j].Name
	})

	got, err := ListCommodities(ctx, db)

	if err != nil {
		t.Errorf("List() returned unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Errorf("List() returned no results")
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}
}

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
