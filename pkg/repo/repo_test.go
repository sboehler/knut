package repo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/pkg/repo/schema"
)

func TestOpenClose(t *testing.T) {
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
}

func TestWriteReadUpdateDeleteAccount(t *testing.T) {
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
	db.GetID()
	db.GetID() // spend a few IDs

	acc := &schema.Account{
		Name:        "foobar",
		AccountType: schema.EXPENSES,
	}
	err = db.Write(func(trx *WriteTrx) error {
		if err := Create(trx, acc); err != nil {
			t.Fatalf("Create(%v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Write(): %v", err)
	}
	var got *schema.Account
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(%v): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("unexpected diff (-want,+got):\n%s\n", diff)
	}
	acc.Name = "barfoo"
	err = db.Write(func(trx *WriteTrx) error {
		err := Update(trx, acc)
		if err != nil {
			t.Fatalf("Update(trx, %v): %v", acc, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Update(): %v", err)
	}
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err != nil {
			t.Fatalf("Read(%v): %v", acc.ID(), err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("unexpected diff (-want,+got):\n%s\n", diff)
	}
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
	err = db.Read(func(trx *ReadTrx) error {
		got, err = Read[schema.Account](trx, acc.ID())
		if err == nil {
			t.Fatalf("Read(%v) returned nil, expected error", acc.ID())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("db.Read(): %v", err)
	}
}
