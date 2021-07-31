package repo

import (
	"context"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateCommodity creates a new commodity.
func CreateCommodity(ctx context.Context, db db, name string) (model.Commodity, error) {
	var (
		row = db.QueryRowContext(ctx,
			`INSERT INTO commodities (name)
			 VALUES (?) 
			 returning id, name`,
			name)
		res model.Commodity
	)
	if row.Err() != nil {
		return res, row.Err()
	}
	return res, row.Scan(&res.ID, &res.Name)
}

// ListCommodities lists all commodities, alphabetically sorted by name.
func ListCommodities(ctx context.Context, db db) ([]model.Commodity, error) {
	var (
		rows, err = db.QueryContext(ctx,
			`SELECT id, name
			 FROM commodities
			 ORDER BY name`,
		)
		res []model.Commodity
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c model.Commodity
		if err = rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, ignoreNoRows(rows.Err())
}
