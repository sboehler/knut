package repo

import "testing"

func TestOpenClose(t *testing.T) {
	d := t.TempDir()

	db, err := Open(d)

	if err != nil {
		t.Fatalf("Open(%s): %v", d, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close(): %v", err)
	}
}
