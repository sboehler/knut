package repo

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CreateVersion() mismatch (-want +got):\n%s", diff)
	}
}

func TestListAccounts(t *testing.T) {
	var (
		t1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2 = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
	)
	var (
		ctx = context.Background()
		db  = createAndMigrateInMemoryDB(ctx, t)
		as  = []model.Account{
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
	_ = populateVersions(ctx, t, db, []string{"v1"})
	var want = populateAccounts(ctx, t, db, as)
	sort.Slice(want, func(i, j int) bool {
		return want[i].Name < want[j].Name
	})
	_ = populateVersions(ctx, t, db, []string{"v1"})

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

func TestUpdateAccounts(t *testing.T) {
	var (
		t1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2 = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
	)
	var (
		ctx = context.Background()
		db  = createAndMigrateInMemoryDB(ctx, t)
		as  = []model.Account{
			{
				Name:     "One",
				OpenDate: t2,
			},
		}
	)
	_ = populateVersions(ctx, t, db, []string{"v1"})
	_ = populateAccounts(ctx, t, db, as)
	_ = populateVersions(ctx, t, db, []string{"v2"})
	var before = listAccounts(ctx, t, db)

	got, err := UpdateAccount(ctx, db, before[0].ID, "Two", t1, &t2)

	if err != nil {
		t.Fatalf("UpdateAccount() returned unexpected error: %v", err)
	}
	var after = listAccounts(ctx, t, db)
	if diff := cmp.Diff(after[0], got); diff != "" {
		t.Errorf("ListAccounts() mismatch (-want +got):\n%s", diff)
	}
}

func TestDeleteAccounts(t *testing.T) {
	var (
		t1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2 = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
	)
	var (
		ctx = context.Background()
		db  = createAndMigrateInMemoryDB(ctx, t)
		as  = []model.Account{
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
	_ = populateVersions(ctx, t, db, []string{"v1"})
	_ = populateAccounts(ctx, t, db, as)
	_ = populateVersions(ctx, t, db, []string{"v1"})
	var before = listAccounts(ctx, t, db)

	ok, err := DeleteAccount(ctx, db, before[0].ID)

	if err != nil {
		t.Fatalf("DeleteAccount() returned unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("DeleteAccount() did not delete an account, but expected it to do so")
	}
	var after = listAccounts(ctx, t, db)
	if diff := cmp.Diff(before[1:], after); diff != "" {
		t.Errorf("ListAccounts() mismatch (-want +got):\n%s", diff)
	}
}

func populateAccounts(ctx context.Context, t *testing.T, db db, accounts []model.Account) []model.Account {
	t.Helper()
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

func listAccounts(ctx context.Context, t *testing.T, db db) []model.Account {
	t.Helper()
	res, err := ListAccounts(ctx, db)
	if err != nil {
		t.Fatalf("ListAccount() returned unexpected error: %v", err)
	}
	return res
}
