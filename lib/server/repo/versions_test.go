package repo

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sboehler/knut/lib/server/model"
)

func TestCreateVersion(t *testing.T) {
	var (
		tests = []struct {
			desc string
			want model.Version
		}{
			{
				desc: "First version",
				want: model.Version{
					Version:     1,
					Description: "First version",
				},
			},
			{
				desc: "Second version",
				want: model.Version{
					Version:     2,
					Description: "Second version",
				},
			},
		}
		ctx = context.Background()
		db  = createAndMigrateInMemoryDB(ctx, t)
	)

	for _, test := range tests {
		got, err := CreateVersion(ctx, db, test.desc)

		if err != nil {
			t.Fatalf("CreateVersion() returned unexpected error: %v", err)
		}
		if diff := cmp.Diff(test.want, got, cmpopts.IgnoreFields(model.Version{}, "CreatedAt")); diff != "" {
			t.Errorf("CreateVersion() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestListVersions(t *testing.T) {
	var (
		ctx   = context.Background()
		db    = createAndMigrateInMemoryDB(ctx, t)
		descs = []string{"CCC", "BBB", "AAA"}
		want  = populateVersions(ctx, t, db, descs)
	)
	sort.Slice(want, func(i, j int) bool {
		return want[i].Version < want[j].Version
	})

	got, err := ListVersions(ctx, db)

	if err != nil {
		t.Fatalf("ListVersions() returned unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Errorf("ListVersions() returned no results")
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListVersions() mismatch (-want +got):\n%s", diff)
	}
}

func TestListVersionsEmpty(t *testing.T) {
	var (
		ctx = context.Background()
		db  = createAndMigrateInMemoryDB(ctx, t)
	)

	got, err := ListVersions(ctx, db)

	if err != nil {
		t.Fatalf("ListVersions() returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("ListVersions() returned unexpected result: %v", got)
	}
}

func TestDeleteVersion(t *testing.T) {
	var (
		ctx   = context.Background()
		db    = createAndMigrateInMemoryDB(ctx, t)
		descs = []string{"CCC", "BBB", "AAA"}
		want  = populateVersions(ctx, t, db, descs)
	)
	sort.Slice(want, func(i, j int) bool {
		return want[i].Version < want[j].Version
	})
	want = want[:len(want)-1]

	var err = DeleteVersion(ctx, db)

	if err != nil {
		t.Fatalf("DeleteVersion() returned unexpected error: %v", err)
	}
	got, err := ListVersions(ctx, db)
	if err != nil {
		t.Fatalf("ListVersions() returned unexpected error: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DeleteVersion() mismatch (-want +got):\n%s", diff)
	}
}

func populateVersions(ctx context.Context, t *testing.T, db db, descriptions []string) []model.Version {
	t.Helper()
	var res []model.Version
	for _, desc := range descriptions {
		c, err := CreateVersion(ctx, db, desc)
		if err != nil {
			t.Fatalf("CreateVersion(ctx, %s) returned unexpected error: %v", desc, err)
		}
		res = append(res, c)
	}
	return res
}
