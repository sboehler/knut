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

func TestWriteReadAccount(t *testing.T) {
	d := t.TempDir()
	db, err := Open(d)
	if err != nil {
		t.Fatalf("Open(%s): %v", d, err)
	}
	acc := &schema.Account{
		Name:        "foobar",
		AccountType: schema.EXPENSES,
	}
	trxW := db.Write()
	if err := trxW.CreateAccount(acc); err != nil {
		t.Fatalf("trxW.CreateAccount(%v): %v", acc, err)
	}
	if err := trxW.trx.Commit(); err != nil {
		t.Fatalf("trxW.trx.Commit(): %v", err)
	}
	trxR := db.Read()
	got, err := trxR.ReadAccount(acc.ID)
	if err != nil {
		t.Fatalf("trxR.ReadAccount(%v): %v", acc.ID, err)
	}
	if diff := cmp.Diff(acc, got); diff != "" {
		t.Fatalf("trxR.ReadAccount(%v): %s", acc.ID, diff)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close(): %v", err)
		}
	})
}
