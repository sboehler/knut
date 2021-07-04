package repo

import (
	"context"
	"database/sql"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateCommodity creates a new commodity.
func CreateCommodity(ctx context.Context, db db, name string) (model.Commodity, error) {
	var (
		row *sql.Row
		res model.Commodity
	)
	if row = db.QueryRowContext(ctx, "INSERT INTO commodities (name) VALUES (?) returning id, name", name); row.Err() != nil {
		return res, row.Err()
	}
	return res, row.Scan(&res.ID, &res.Name)
}

// ListCommodities lists all commodities, alphabetically sorted by name.
func ListCommodities(ctx context.Context, db db) ([]model.Commodity, error) {
	var (
		rows *sql.Rows
		err  error
	)
	rows, err = db.QueryContext(ctx, "SELECT id, name FROM commodities ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []model.Commodity
	for rows.Next() {
		var c model.Commodity
		if err = rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, rows.Err()
}
