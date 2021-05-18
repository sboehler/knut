package repo

import (
	"context"
	"database/sql"
	"time"
)

type db interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func parseDatetime(s string, dest *time.Time) error {
	dt, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*dest = dt
	return nil
}
