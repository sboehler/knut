package repo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/pkg/repo/db"
)

func setupDB(t *testing.T) *db.DB {
	t.Helper()
	td := t.TempDir()
	d, err := db.Open(td)
	if err != nil {
		t.Fatalf("Open(%s): %v", td, err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Fatalf("d.Close(): %v", err)
		}
	})
	d.GetID() // spend a few IDs
	d.GetID()
	return d
}

func TestWriteReadUpdateDeleteAccount(t *testing.T) {
	d := setupDB(t)
	acc := &Account{
		Name:        "foobar",
		AccountType: EXPENSES,
	}
	// create
	err := d.Write(func(trx *db.WriteTrx) error {
		if err := AccountTable.Create(trx, acc); err != nil {
			t.Fatalf("Create(trx, %v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Write(): %v", err)
	}
	// read
	var got *Account
	err = d.Read(func(trx *db.ReadTrx) error {
		got, err = AccountTable.Read(trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(trx, %v): %d", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("d.Read(trx, %d): unexpected diff (-want,+got):\n%s\n", acc.ID(), diff)
	}
	// update
	acc.Name = "barfoo"
	err = d.Write(func(trx *db.WriteTrx) error {
		err := AccountTable.Update(trx, acc)
		if err != nil {
			t.Fatalf("Update(trx, %v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Write(): %v", err)
	}
	// read
	err = d.Read(func(trx *db.ReadTrx) error {
		got, err = AccountTable.Read(trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(trx, %v): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("unexpected diff (-want,+got):\n%s\n", diff)
	}
	// delete
	err = d.Write(func(trx *db.WriteTrx) error {
		err := AccountTable.Delete(trx, acc.ID())
		if err != nil {
			t.Fatalf("Delete(trx, %d): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Delete(): %v", err)
	}
	// read
	err = d.Read(func(trx *db.ReadTrx) error {
		got, err = AccountTable.Read(trx, acc.ID())
		if err == nil {
			t.Fatalf("Read(trx, %v) = %v, nil, want nil, err", acc.ID(), got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("d.Read(): %v", err)
	}
}
