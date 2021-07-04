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
		account model.Account
		row     *sql.Row
		err     error
	)
	row = db.QueryRowContext(ctx, `INSERT INTO account_ids DEFAULT VALUES RETURNING id`)
	if row.Err() != nil {
		return account, row.Err()
	}
	if err = row.Scan(&account.ID); err != nil {
		return account, err
	}
	row = db.QueryRowContext(ctx,
		`INSERT INTO accounts(id, name, open_date, close_date) VALUES (?, ?, ?, ?)
		returning id, name, datetime(open_date), datetime(close_date)`,
		account.ID, name, openDate, closeDate)
	if row.Err() != nil {
		return account, row.Err()
	}
	return account, rowToAccount(row, &account)
}

// ListAccounts lists all accounts.
func ListAccounts(ctx context.Context, db db) ([]model.Account, error) {
	var res []model.Account
	rows, err := db.QueryContext(ctx, `
	  SELECT id, name, datetime(open_date), datetime(close_date) 
	  FROM accounts 
	  ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var account model.Account
		if err := rowToAccount(rows, &account); err != nil {
			return nil, err
		}
		res = append(res, account)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

// UpdateAccount updates an account in the current version.
func UpdateAccount(ctx context.Context, db db, id model.AccountID, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		row     *sql.Row
		account model.Account
	)
	row = db.QueryRowContext(ctx,
		`UPDATE accounts
		SET name = ?, open_date = ?, close_date = ?
		WHERE id = ?
		RETURNING id, name, datetime(open_date), datetime(close_date)`, name, openDate, closeDate, id)
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
	_, err := db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func rowToAccount(row scan, res *model.Account) error {
	var (
		err       error
		openDate  string
		closeDate sql.NullString
	)
	if err = row.Scan(&res.ID, &res.Name, &openDate, &closeDate); err != nil {
		return err
	}
	if err = parseDatetime(openDate, &res.OpenDate); err != nil {
		return err
	}
	if closeDate.Valid {
		var t time.Time
		if err = parseDatetime(closeDate.String, &t); err != nil {
			return err
		}
		res.CloseDate = &t
	}
	return nil
}
