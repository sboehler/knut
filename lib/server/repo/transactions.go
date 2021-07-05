package repo

import (
	"context"
	"database/sql"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateTransaction creates transaction.
func CreateTransaction(ctx context.Context, db db, t model.Transaction) (model.Transaction, error) {
	var (
		row *sql.Row
		err error
	)
	if row = db.QueryRowContext(ctx, `INSERT INTO transaction_ids DEFAULT VALUES RETURNING id`); row.Err() != nil {
		return t, row.Err()
	}
	if err = row.Scan(&t.ID); err != nil {
		return t, err
	}
	if row = db.QueryRowContext(ctx,
		`INSERT INTO transactions_history(id, date, description) VALUES (?, ?, ?) returning id, datetime(date), description`,
		t.ID, t.Date, t.Description); row.Err() != nil {
		return t, row.Err()
	}
	var res model.Transaction
	if err = rowToTrx(row, &res); err != nil {
		return t, err
	}
	for _, b := range t.Bookings {
		var resB model.Booking
		if resB, err = insertBooking(ctx, db, t.ID, b); err != nil {
			return t, err
		}
		res.Bookings = append(res.Bookings, resB)
	}
	return res, nil
}

// UpdateTransaction updates a transaction.
func UpdateTransaction(ctx context.Context, db db, t model.Transaction) (model.Transaction, error) {
	var (
		row *sql.Row
		res model.Transaction
		err error
	)
	if row = db.QueryRowContext(ctx,
		`UPDATE transactions SET date = ?, description = ?
		WHERE id = ?
		RETURNING id, datetime(date), description`, t.Date, t.Description, t.ID); row.Err() != nil {
		return res, row.Err()
	}
	if err = rowToTrx(row, &res); err != nil {
		return res, err
	}
	for _, b := range t.Bookings {
		var resB model.Booking
		if resB, err = insertBooking(ctx, db, t.ID, b); err != nil {
			return res, err
		}
		res.Bookings = append(res.Bookings, resB)
	}
	return res, nil
}

// ListTransactions fetches all transactions.
func ListTransactions(ctx context.Context, db db) ([]model.Transaction, error) {
	bookings, err := ListBookings(ctx, db)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, datetime(date), description 
		FROM transactions 
		ORDER BY date, description, id`)
	if err != nil {
		return nil, err
	}
	var res []model.Transaction
	defer rows.Close()
	for rows.Next() {
		var t model.Transaction
		if err := rowToTrx(rows, &t); err != nil {
			return nil, err
		}
		t.Bookings = bookings[t.ID]
		res = append(res, t)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

// ListBookings lists all bookings.
func ListBookings(ctx context.Context, db db) (map[model.TransactionID][]model.Booking, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, amount, commodity_id, credit_account_id, debit_account_id FROM bookings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res = make(map[model.TransactionID][]model.Booking)
	for rows.Next() {
		var b model.Booking
		if err := rowToBooking(rows, &b); err != nil {
			return nil, err
		}
		res[b.ID] = append(res[b.ID], b)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

func insertBooking(ctx context.Context, db db, tid model.TransactionID, b model.Booking) (model.Booking, error) {
	var row *sql.Row
	if row = db.QueryRowContext(ctx,
		`INSERT INTO bookings(id, amount, commodity_id, credit_account_id, debit_account_id) VALUES (?, ?, ?, ?, ?)
		returning id, amount, commodity_id, credit_account_id, debit_account_id`,
		tid, b.Amount, b.CommodityID, b.CreditAccountID, b.DebitAccountID); row.Err() != nil {
		return b, row.Err()
	}
	var res model.Booking
	if err := rowToBooking(row, &res); err != nil {
		return res, err
	}
	return res, nil
}

func rowToTrx(row scan, t *model.Transaction) error {
	var d string
	if err := row.Scan(&t.ID, &d, &t.Description); err != nil {
		return err
	}
	return parseDatetime(d, &t.Date)
}

func rowToBooking(row scan, res *model.Booking) error {
	return row.Scan(&res.ID, &res.Amount, &res.CommodityID, &res.CreditAccountID, &res.DebitAccountID)
}
