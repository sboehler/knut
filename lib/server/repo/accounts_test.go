package repo

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/server/model"
)

func TestCreateAccount(t *testing.T) {
	var (
		d        = date.Date(2020, time.December, 31)
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Accounts: []model.Account{
				{Name: "MyAccount", OpenDate: date.Date(2021, time.May, 14)},
			},
		})
		tests = []struct {
			desc      string
			name      string
			openDate  time.Time
			closeDate *time.Time
		}{
			{
				desc:      "new account",
				name:      "Another account",
				openDate:  date.Date(2021, 7, 31),
				closeDate: &d,
			},
			{
				desc:      "duplicate account",
				name:      "MyAccount",
				openDate:  date.Date(2021, 6, 31),
				closeDate: &d,
			},
			{
				desc:     "account without close date",
				name:     "MyAccount",
				openDate: date.Date(2021, 6, 31),
			},
		}
	)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := beginTransaction(ctx, t, db)
			defer db.Rollback()

			c, err := CreateAccount(ctx, db, test.name, test.openDate, test.closeDate)
			if err != nil {
				t.Fatalf("CreateAccount returned unexpected error %v", err)
			}
			var (
				want = Scenario{Accounts: append(scenario.Accounts, c)}
				got  = Load(ctx, t, db)
			)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("CreateAccount() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestListAccounts(t *testing.T) {
	var (
		t1 = date.Date(2021, time.May, 14)
		t2 = date.Date(2022, time.May, 14)
	)
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Accounts: []model.Account{
				{Name: "Account1", OpenDate: t1},
				{Name: "Account2", OpenDate: t1, CloseDate: &t2},
				{Name: "Account3", OpenDate: t2},
			},
		})
	)

	got, err := ListAccounts(ctx, db)

	if err != nil {
		t.Fatalf("ListAccounts() returned unexpected error: %v", err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Less(got[j]) })
	if diff := cmp.Diff(scenario.Accounts, got); diff != "" {
		t.Errorf("ListAccounts() mismatch (-want +got):\n%s", diff)
	}
}

func TestUpdateAccounts(t *testing.T) {
	var (
		t1       = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2       = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Accounts: []model.Account{
				{Name: "Account1", OpenDate: t1},
				{Name: "Account2", OpenDate: t1, CloseDate: &t2},
				{Name: "Account3", OpenDate: t2},
			},
		})
		tests = []struct {
			desc   string
			update model.Account
		}{
			{
				desc: "update name",
				update: model.Account{
					ID: scenario.Accounts[1].ID, Name: "New Name", OpenDate: t1, CloseDate: nil},
			},
		}
	)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			db := beginTransaction(ctx, t, db)
			defer db.Rollback()

			a, err := UpdateAccount(ctx, db, test.update.ID, test.update.Name, test.update.OpenDate, test.update.CloseDate)

			if err != nil {
				t.Fatalf("UpdateAccount() returned unexpected error: %v", err)
			}
			if a.ID != test.update.ID || a.Name != test.update.Name || a.OpenDate != test.update.OpenDate || a.CloseDate != test.update.CloseDate {
				t.Fatalf("UpdateAccount() did not update account, want = %v, got = %v", test.update, a)
			}
			var (
				got  = Load(ctx, t, db)
				want = scenario.DeepCopy()
			)
			for i, acc := range want.Accounts {
				if acc.ID == test.update.ID {
					want.Accounts[i] = test.update
				}
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("UpdateAccounts() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteAccounts(t *testing.T) {
	var (
		t1 = time.Date(2021, time.May, 14, 0, 0, 0, 0, time.UTC)
		t2 = time.Date(2022, time.May, 14, 0, 0, 0, 0, time.UTC)
	)
	var (
		ctx      = context.Background()
		db       = createAndMigrateInMemoryDB(ctx, t)
		scenario = Save(ctx, t, db, Scenario{
			Accounts: []model.Account{
				{Name: "Account1", OpenDate: t1},
				{Name: "Account2", OpenDate: t1, CloseDate: &t2},
				{Name: "Account3", OpenDate: t2},
			},
		})
	)

	for i, acc := range scenario.Accounts {
		t.Run(fmt.Sprintf("delete account %d", i), func(t *testing.T) {
			db := beginTransaction(ctx, t, db)
			defer db.Rollback()

			err := DeleteAccount(ctx, db, acc.ID)

			if err != nil {
				t.Fatalf("DeleteAccount returned unexpected error %v", err)
			}
			var (
				got  = Load(ctx, t, db)
				want = Scenario{}
			)
			for _, a := range scenario.Accounts {
				if a.ID != acc.ID {
					want.Accounts = append(want.Accounts, a)
				}
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("DeleteAccounts() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
