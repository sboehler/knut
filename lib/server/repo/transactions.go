package repo

import (
	"context"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateTransaction creates transaction.
func CreateTransaction(ctx context.Context, db db, t model.Transaction) (model.Transaction, error) {
	var (
		res model.Transaction
		err error
	)
	if t.ID, err = createTransactionID(ctx, db); err != nil {
		return res, err
	}
	if res, err = createTransaction(ctx, db, t.ID, t.Date, t.Description); err != nil {
		return res, err
	}
	res.Bookings, err = createBookings(ctx, db, t.ID, t.Bookings)
	return res, err
}

func createTransactionID(ctx context.Context, db db) (model.TransactionID, error) {
	var (
		row = db.QueryRowContext(ctx,
			`INSERT INTO transaction_ids
			 DEFAULT VALUES
			 RETURNING id`,
		)
		res model.TransactionID
	)
	if row.Err() != nil {
		return res, row.Err()
	}
	return res, row.Scan(&res)
}

func createTransaction(ctx context.Context, db db, id model.TransactionID, date time.Time, desc string) (model.Transaction, error) {
	var row = db.QueryRowContext(ctx,
		`INSERT INTO transactions_history(id, date, description) 
			 VALUES (?, ?, ?)
			 RETURNING id, datetime(date), description`,
		id, date, desc)
	if row.Err() != nil {
		return model.Transaction{}, row.Err()
	}
	return rowToTrx(row)
}

// UpdateTransaction updates a transaction.
func UpdateTransaction(ctx context.Context, db db, t model.Transaction) (model.Transaction, error) {
	var (
		res model.Transaction
		err error
	)
	if res, err = updateTransaction(ctx, db, t.ID, t.Date, t.Description); err != nil {
		return res, err
	}
	res.Bookings, err = createBookings(ctx, db, t.ID, t.Bookings)
	return res, err
}

func updateTransaction(ctx context.Context, db db, id model.TransactionID, date time.Time, desc string) (model.Transaction, error) {
	var row = db.QueryRowContext(ctx,
		`UPDATE transactions
		 SET date = ?, description = ?
		 WHERE id = ?
		 RETURNING id, datetime(date), description`,
		date, desc, id)
	if row.Err() != nil {
		return model.Transaction{}, row.Err()
	}
	return rowToTrx(row)
}

// ListTransactions fetches all transactions.
func ListTransactions(ctx context.Context, db db) ([]model.Transaction, error) {
	var (
		bookings          map[model.TransactionID][]model.Booking
		transactions, res []model.Transaction
		err               error
	)
	if transactions, err = listTransactions(ctx, db); err != nil {
		return nil, err
	}
	if bookings, err = listBookings(ctx, db); err != nil {
		return nil, err
	}
	for _, t := range transactions {
		t.Bookings = bookings[t.ID]
		res = append(res, t)
	}
	return res, nil
}

func listTransactions(ctx context.Context, db db) ([]model.Transaction, error) {
	var (
		rows, err = db.QueryContext(ctx,
			`SELECT id, datetime(date), description 
			 FROM transactions 
			 ORDER BY date, description, id`)
		res []model.Transaction
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t model.Transaction
		if t, err = rowToTrx(rows); err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, ignoreNoRows(rows.Err())
}

// listBookings lists all bookings.
func listBookings(ctx context.Context, db db) (map[model.TransactionID][]model.Booking, error) {
	var (
		rows, err = db.QueryContext(ctx,
			`SELECT id, amount, commodity_id, credit_account_id, debit_account_id
			 FROM bookings`,
		)
		res = make(map[model.TransactionID][]model.Booking)
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			b  model.Booking
			id model.TransactionID
		)
		if id, b, err = rowToBooking(rows); err != nil {
			return nil, err
		}
		res[id] = append(res[id], b)
	}
	return res, ignoreNoRows(rows.Err())
}

func createBookings(ctx context.Context, db db, id model.TransactionID, bs []model.Booking) ([]model.Booking, error) {
	var res []model.Booking
	for _, b := range bs {
		var (
			row = db.QueryRowContext(ctx,
				`INSERT INTO bookings(id, amount, commodity_id, credit_account_id, debit_account_id)
				 VALUES (?, ?, ?, ?, ?)
				 RETURNING id, amount, commodity_id, credit_account_id, debit_account_id`,
				id, b.Amount, b.CommodityID, b.CreditAccountID, b.DebitAccountID)
			err error
		)
		if row.Err() != nil {
			return nil, row.Err()
		}
		_, b, err = rowToBooking(row)
		if err != nil {
			return nil, err
		}
		res = append(res, b)
	}
	return res, nil
}

func rowToTrx(row scan) (model.Transaction, error) {
	var (
		s   string
		res model.Transaction
		err error
	)
	if err = row.Scan(&res.ID, &s, &res.Description); err != nil {
		return res, err
	}
	if res.Date, err = parseDatetime(s); err != nil {
		return res, err
	}
	return res, nil
}

func rowToBooking(row scan) (model.TransactionID, model.Booking, error) {
	var (
		res model.Booking
		id  model.TransactionID
	)
	return id, res, row.Scan(&id, &res.Amount, &res.CommodityID, &res.CreditAccountID, &res.DebitAccountID)
}
