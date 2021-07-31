package repo

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// InsertPrice creates a price.
func InsertPrice(ctx context.Context, db db, price model.Price) (model.Price, error) {
	var (
		row = db.QueryRowContext(ctx,
			`INSERT INTO prices(date, commodity_id, target_commodity_id, price) 
			 VALUES (?, ?, ?, ?)
			 RETURNING datetime(date), commodity_id, target_commodity_id, price`,
			price.Date, price.CommodityID, price.TargetCommodityID, price.Price,
		)
		res model.Price
	)
	if row.Err() != nil {
		return res, row.Err()
	}
	return rowToPrice(row)
}

// ListPrices lists all prices.
func ListPrices(ctx context.Context, db db) ([]model.Price, error) {
	var (
		rows, err = db.QueryContext(ctx,
			`SELECT datetime(date), commodity_id, target_commodity_id, price 
			 FROM prices 
		     ORDER BY date, commodity_id, target_commodity_id`,
		)
		res []model.Price
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var price model.Price
		if price, err = rowToPrice(rows); err != nil {
			return nil, err
		}
		res = append(res, price)
	}
	return res, ignoreNoRows(err)
}

// DeletePrice deletes a price.
func DeletePrice(ctx context.Context, db db, date time.Time, commodityID, targetCommodityID model.CommodityID) error {
	_, err := db.ExecContext(ctx,
		`DELETE
		 FROM prices
		 WHERE date = ? AND commodity_id = ? AND target_commodity_id = ?`,
		date, commodityID, targetCommodityID,
	)
	return err
}

func rowToPrice(row scan) (model.Price, error) {
	var (
		s   string
		res model.Price
		err error
	)
	if err = row.Scan(&s, &res.CommodityID, &res.TargetCommodityID, &res.Price); err != nil {
		return res, err
	}
	res.Date, err = parseDatetime(s)
	return res, err
}
