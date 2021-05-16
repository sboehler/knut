package repo

import (
	"context"
	"database/sql"

	"github.com/sboehler/knut/lib/server/model"
)

type db interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// CreateCommodity creates a new commodity.
func CreateCommodity(ctx context.Context, db db, name string) (model.Commodity, error) {
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

// ListCommodities lists all commodities, alphabetically sorted by name.
func ListCommodities(ctx context.Context, db db) ([]model.Commodity, error) {
	var res []model.Commodity
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM commodities ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c model.Commodity
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, rows.Err()
}
