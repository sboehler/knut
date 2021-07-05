package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// InsertPrice creates a price.
func InsertPrice(ctx context.Context, db db, price model.Price) (model.Price, error) {
	var (
		res model.Price
		err error
	)
	row := db.QueryRowContext(ctx,
		`INSERT INTO prices(date, commodity_id, target_commodity_id, price) 
		VALUES (?, ?, ?, ?)
		RETURNING datetime(date), commodity_id, target_commodity_id, price`,
		price.Date, price.CommodityID, price.TargetCommodityID, price.Price)
	if row.Err() != nil {
		return res, row.Err()
	}
	if err = rowToPrice(row, &res); err != nil {
		return res, err
	}
	return res, nil
}

// ListPrices lists all prices.
func ListPrices(ctx context.Context, db db) ([]model.Price, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT datetime(date), commodity_id, target_commodity_id, price 
		FROM prices 
	    ORDER BY date, commodity_id, target_commodity_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []model.Price
	for rows.Next() {
		var price model.Price
		if err := rowToPrice(rows, &price); err != nil {
			return nil, err
		}
		res = append(res, price)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

// DeletePrice deletes a price.
func DeletePrice(ctx context.Context, db db, date time.Time, commodityID, targetCommodityID model.CommodityID) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM prices WHERE date = ? AND commodity_id = ? and target_commodity_id = ?`,
		date, commodityID, targetCommodityID)
	return err
}

func rowToPrice(row scan, res *model.Price) error {
	var s string
	if err := row.Scan(&s, &res.CommodityID, &res.TargetCommodityID, &res.Price); err != nil {
		return err
	}
	return parseDatetime(s, &res.Date)
}