package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// InsertPrice creates a price.
func InsertPrice(ctx context.Context, db db, price model.Price) error {
	var err error
	_, err = db.ExecContext(ctx,
		`INSERT INTO prices(date, commodity_id, target_commodity_id, price) 
		VALUES (?, ?, ?, ?)`,
		price.Date, price.CommodityID, price.TargetCommodityID, price.Price)
	return err
}

// ListPrices lists all prices.
func ListPrices(ctx context.Context, db db) ([]model.Price, error) {
	var res []model.Price
	rows, err := db.QueryContext(ctx,
		`SELECT datetime(date), commodity_id, target_commodity_id, price 
	  	FROM prices 
	    ORDER BY date, commodity_id, target_commodity_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		price, err := rowToPrice(rows)
		if err != nil {
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
func DeletePrice(ctx context.Context, db db, date time.Time, commodityID, targetCommodityID int) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM prices WHERE date = ? AND commodity_id = ? and target_commodity_id = ?`,
		date, commodityID, targetCommodityID)
	return err
}

func rowToPrice(row scan) (model.Price, error) {
	var (
		res model.Price
		err error
		s   string
	)
	if err = row.Scan(&s, &res.CommodityID, &res.TargetCommodityID, &res.Price); err != nil {
		return res, err
	}
	if err = parseDatetime(s, &res.Date); err != nil {
		return res, err
	}
	return res, nil
}
