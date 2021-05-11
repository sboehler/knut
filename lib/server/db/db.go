package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql
var migrations embed.FS

type Handle struct {
	DB *sql.DB
}

func Open(path string) (*Handle, error) {
	var (
		db  *sql.DB
		err error
	)
	db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, err
	}
	return &Handle{db}, nil
}

func (h *Handle) Migrate(ctx context.Context) error {
	conn, err := h.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var version int
	if err := conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	files, err := migrations.ReadDir("sql")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	for _, f := range files {
		nbr := f.Name()[:3]
		i, err := strconv.Atoi(nbr)
		if err != nil {
			return err
		}
		if i <= version {
			continue
		}
		f, err := migrations.ReadFile(path.Join("sql", f.Name()))
		if err != nil {
			return err
		}
		txn, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := txn.ExecContext(ctx, string(f)); err != nil {
			return err
		}
		if _, err := txn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i)); err != nil {
			return err
		}
		if err := txn.Commit(); err != nil {
			return err
		}
	}
	return nil
}
