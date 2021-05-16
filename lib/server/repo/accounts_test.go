package repo

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/server/model"
)

func TestCreateAccount(t *testing.T) {
	var (
		name     = "MyAccount"
		openDate = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		want     = model.Account{
			ID:       1,
			Name:     name,
			OpenDate: openDate,
		}
	)
	_ = populateVersions(ctx, t, db, []string{"first version"})

	got, err := CreateAccount(ctx, db, name, openDate, nil)

	if err != nil {
		t.Fatalf("CreateVersion() returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(model.Account{}, "CreatedAt")); diff != "" {
		t.Errorf("CreateVersion() mismatch (-want +got):\n%s", diff)
	}
}

func TestListAccounts(t *testing.T) {
	var (
		t1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2 = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
	)
	var (
		ctx   = context.Background()
		db    = createAndMigrateInMemoryDB(ctx, t)
		descs = []string{"CCC", "BBB", "AAA"}
		as    = []model.Account{
			{
				Name:     "One",
				OpenDate: t2,
			},
			{
				Name:      "Two",
				OpenDate:  t1,
				CloseDate: &t2,
			},
		}
	)
	_ = populateVersions(ctx, t, db, descs)
	var want = populateAccounts(ctx, t, db, as)
	sort.Slice(want, func(i, j int) bool {
		return want[i].Name < want[j].Name
	})

	got, err := ListAccounts(ctx, db)

	if err != nil {
		t.Fatalf("ListAccounts() returned unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Errorf("ListAccounts() returned no results")
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListAccounts() mismatch (-want +got):\n%s", diff)
	}
}

func populateAccounts(ctx context.Context, t *testing.T, db db, accounts []model.Account) []model.Account {
	var res []model.Account
	for _, account := range accounts {
		a, err := CreateAccount(ctx, db, account.Name, account.OpenDate, account.CloseDate)
		if err != nil {
			t.Fatalf("CreateAccount() returned unexpected error: %v", err)
		}
		res = append(res, a)
	}
	return res
}
