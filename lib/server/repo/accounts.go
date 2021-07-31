package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateAccount creates an account in the current version.
func CreateAccount(ctx context.Context, db db, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		id  model.AccountID
		err error
	)
	if id, err = createAccountID(ctx, db); err != nil {
		return model.Account{}, err
	}
	return createAccount(ctx, db, id, name, openDate, closeDate)
}

func createAccountID(ctx context.Context, db db) (model.AccountID, error) {
	var (
		row = db.QueryRowContext(ctx,
			`INSERT INTO account_ids DEFAULT VALUES RETURNING id`,
		)
		res model.AccountID
	)
	if row.Err() != nil {
		return res, row.Err()
	}
	return res, row.Scan(&res)
}

func createAccount(ctx context.Context, db db, id model.AccountID, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		row = db.QueryRowContext(ctx,
			`INSERT INTO accounts(id, name, open_date, close_date) 
			 VALUES (?, ?, ?, ?)
			 RETURNING id, name, datetime(open_date), datetime(close_date)`,
			id, name, openDate, closeDate)
		account model.Account
	)
	if row.Err() != nil {
		return account, row.Err()
	}
	return account, rowToAccount(row, &account)
}

// ListAccounts lists all accounts.
func ListAccounts(ctx context.Context, db db) ([]model.Account, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, datetime(open_date), datetime(close_date) 
		 FROM accounts 
		 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []model.Account
	for rows.Next() {
		var account model.Account
		if err := rowToAccount(rows, &account); err != nil {
			return nil, err
		}
		res = append(res, account)
	}
	return res, ignoreNoRows(rows.Err())
}

// UpdateAccount updates an account in the current version.
func UpdateAccount(ctx context.Context, db db, id model.AccountID, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		row = db.QueryRowContext(ctx,
			`UPDATE accounts
			 SET name = ?, open_date = ?, close_date = ?
			 WHERE id = ?
			 RETURNING id, name, datetime(open_date), datetime(close_date)`,
			name, openDate, closeDate, id)
		account model.Account
	)
	if row.Err() != nil {
		return account, row.Err()
	}
	return account, rowToAccount(row, &account)
}

type scan interface {
	Scan(dest ...interface{}) error
}

// DeleteAccount deletes an account in the current version.
func DeleteAccount(ctx context.Context, db db, id model.AccountID) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM accounts
		 WHERE id = ?`,
		id)
	return err
}

func rowToAccount(row scan, res *model.Account) error {
	var (
		openDate  string
		closeDate sql.NullString
		err       error
	)
	if err = row.Scan(&res.ID, &res.Name, &openDate, &closeDate); err != nil {
		return err
	}
	if res.OpenDate, err = parseDatetime(openDate); err != nil {
		return err
	}
	if closeDate.Valid {
		var t time.Time
		if t, err = parseDatetime(closeDate.String); err != nil {
			return err
		}
		res.CloseDate = &t
	}
	return nil
}
