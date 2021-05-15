package repo

import (
	"context"
	"database/sql"

	"github.com/sboehler/knut/lib/server/model"
)

type database interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func Create(ctx context.Context, db database, name string) (model.Commodity, error) {
	var (
		row *sql.Row
		res = model.Commodity{
			Name: name,
		}
	)
	if row = db.QueryRowContext(ctx, "INSERT INTO commodities (name) VALUES (?) returning id", name); row.Err() != nil {
		return res, row.Err()
	}
	return res, row.Scan(&res.ID)
}
