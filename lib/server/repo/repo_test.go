package repo

import (
	"context"
	"testing"

	"github.com/sboehler/knut/lib/server/db"
)

func TestCreateCommodity(t *testing.T) {
	var ctx = context.Background()
	h, err := db.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("error creating in-memory database: %v", err)
	}
	name := "CHF"

	c, err := Create(ctx, h, name)

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
