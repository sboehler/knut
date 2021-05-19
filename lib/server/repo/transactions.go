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
	row = db.QueryRowContext(ctx, `INSERT INTO transaction_ids DEFAULT VALUES RETURNING id`)
	if row.Err() != nil {
		return t, row.Err()
	}
	if err = row.Scan(&t.ID); err != nil {
		return t, err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO transactions_history(id, date, description) VALUES (?, ?, ?)`,
		t.ID, t.Date, t.Description)
	if err != nil {
		return t, err
	}
	for _, b := range t.Bookings {
		_, err = db.ExecContext(ctx,
			`INSERT INTO bookings_history(id, amount, commodity_id, credit_account_id, debit_account_id) VALUES (?, ?, ?, ?, ?)`,
			t.ID, b.Amount, b.CommodityID, b.CreditAccountID, b.DebitAccountID)
		if err != nil {
			return t, err
		}
	}
	return t, nil
}

// ListTransactions fetches all transactions.
func ListTransactions(ctx context.Context, db db) ([]model.Transaction, error) {
	var res []model.Transaction
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
	defer rows.Close()
	for rows.Next() {
		t, err := rowToTrx(rows)
		if err != nil {
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
func ListBookings(ctx context.Context, db db) (map[int][]model.Booking, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, amount, commodity_id, credit_account_id, debit_account_id FROM bookings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res = make(map[int][]model.Booking)
	for rows.Next() {
		b, err := rowToBooking(rows)
		if err != nil {
			return nil, err
		}
		res[b.ID] = append(res[b.ID], b)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

func rowToTrx(row scan) (model.Transaction, error) {
	var (
		res  model.Transaction
		err  error
		date string
	)
	if err = row.Scan(&res.ID, &date, &res.Description); err != nil {
		return res, err
	}
	return res, parseDatetime(date, &res.Date)
}

func rowToBooking(row scan) (res model.Booking, err error) {
	return res, row.Scan(&res.ID, &res.Amount, &res.CommodityID, &res.CreditAccountID, &res.DebitAccountID)
}
