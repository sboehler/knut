package repo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/pkg/repo/schema"
)

func setupDB(t *testing.T) *DB {
	t.Helper()
	d := t.TempDir()
	db, err := Open(d)
	if err != nil {
		t.Fatalf("Open(%s): %v", d, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close(): %v", err)
		}
	})
	db.GetID() // spend a few IDs
	db.GetID()
	return db
}

func TestWriteReadUpdateDeleteAccount(t *testing.T) {
	db := setupDB(t)
	acc := &schema.Account{
		Name:        "foobar",
		AccountType: schema.EXPENSES,
	}
	// create
	err := db.Write(func(trx *WriteTrx) error {
		if err := Create(trx, acc); err != nil {
			t.Fatalf("Create(trx, %v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Write(): %v", err)
	}
	// read
	var got *schema.Account
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(trx, %v): %d", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("db.Read(trx, %d): unexpected diff (-want,+got):\n%s\n", acc.ID(), diff)
	}
	// update
	acc.Name = "barfoo"
	err = db.Write(func(trx *WriteTrx) error {
		err := Update(trx, acc)
		if err != nil {
			t.Fatalf("Update(trx, %v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Write(): %v", err)
	}
	// read
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(trx, %v): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("unexpected diff (-want,+got):\n%s\n", diff)
	}
	// delete
	err = db.Write(func(trx *WriteTrx) error {
		err := Delete[schema.Account](trx, acc.ID())
		if err != nil {
			t.Fatalf("Delete(trx, %d): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Delete(): %v", err)
	}
	// read
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err == nil {
			t.Fatalf("Read(trx, %v) = %v, nil, want nil, err", acc.ID(), got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
}
