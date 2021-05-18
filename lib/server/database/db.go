package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"

	// use SQLite3
	_ "github.com/mattn/go-sqlite3"
)

//go:embed sql idem
var migrations embed.FS

// Open opens and migrate an SQLite3 database.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	var (
		db  *sql.DB
		err error
	)
	db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
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
		s, err := migrations.ReadFile(path.Join("sql", f.Name()))
		if err != nil {
			return err
		}
		txn, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := txn.ExecContext(ctx, string(s)); err != nil {
			return err
		}
		if _, err := txn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i)); err != nil {
			return err
		}
		if err := txn.Commit(); err != nil {
			return err
		}
	}
	files, err = migrations.ReadDir("idem")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	for _, f := range files {
		s, err := migrations.ReadFile(path.Join("idem", f.Name()))
		if err != nil {
			return err
		}
		txn, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := txn.ExecContext(ctx, string(s)); err != nil {
			return err
		}
		if err := txn.Commit(); err != nil {
			return err
		}
	}
	return nil
}
