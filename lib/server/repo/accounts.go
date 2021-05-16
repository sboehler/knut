package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateAccount creates an account in the current version.
func CreateAccount(ctx context.Context, db db, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		row *sql.Row
		res = model.Account{}
		err error
	)
	if row = db.QueryRowContext(ctx, `INSERT INTO account_ids DEFAULT VALUES RETURNING id`); row.Err() != nil {
		return res, row.Err()
	}
	if err = row.Scan(&res.ID); err != nil {
		return res, err
	}
	row = db.QueryRowContext(ctx,
		`INSERT INTO accounts(account_id, name, open_date, close_date, version_from, version_to)
		SELECT ?, ?, ?, ?, (SELECT MAX(version) FROM versions), 9223372036854775807
		RETURNING name, datetime(open_date), datetime(close_date)`, res.ID, name, openDate, closeDate)
	if row.Err() != nil {
		return res, row.Err()
	}
	var (
		os string
		cs sql.NullString
	)
	if err = row.Scan(&res.Name, &os, &cs); err != nil {
		return res, err
	}
	if err = parseDatetime(os, &res.OpenDate); err != nil {
		return res, err
	}
	if cs.Valid {
		var t time.Time
		if err = parseDatetime(cs.String, &t); err != nil {
			return res, err
		}
		res.CloseDate = &t
	}
	return res, nil
}

// ListAccounts lists all accounts.
func ListAccounts(ctx context.Context, db db) ([]model.Account, error) {
	var res []model.Account
	rows, err := db.QueryContext(ctx, `
	  SELECT account_id, name, datetime(open_date), datetime(close_date) 
	  FROM accounts
	  WHERE version_from <= (SELECT MAX(version) from versions) AND
	  version_to > (SELECT MAX(version) from versions) 
	  ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			c  model.Account
			os string
			cs sql.NullString
		)
		if err := rows.Scan(&c.ID, &c.Name, &os, &cs); err != nil {
			return nil, err
		}
		if err = parseDatetime(os, &c.OpenDate); err != nil {
			return nil, err
		}
		if cs.Valid {
			var t time.Time
			if err = parseDatetime(cs.String, &t); err != nil {
				return res, err
			}
			c.CloseDate = &t
		}
		res = append(res, c)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

// UpdateAccount updates an account in the current version.
func UpdateAccount(ctx context.Context, db db, id int, name string, openDate time.Time, closeDate *time.Time) (model.Account, error) {
	var (
		row *sql.Row
		res = model.Account{}
		err error
	)
	rslt, err := db.ExecContext(ctx, `
	  UPDATE accounts 
	  SET version_to = (SELECT MAX(version) FROM versions)
	  WHERE account_id = ? AND version_to = 9223372036854775807`, id)
	if err != nil {
		return res, err
	}
	n, err := rslt.RowsAffected()
	if err != nil {
		return res, err
	}
	if n != 1 {
		return res, fmt.Errorf("UpdateAccount() affected %d rows, expected 1", n)
	}
	row = db.QueryRowContext(ctx,
		`INSERT INTO accounts(account_id, name, open_date, close_date, version_from, version_to)
		SELECT ?, ?, ?, ?, (SELECT MAX(version) FROM versions), 9223372036854775807
		ON CONFLICT (account_id, version_from, version_to) DO UPDATE
		SET name = excluded.name, open_date = excluded.open_date, close_date = excluded.close_date
		RETURNING name, datetime(open_date), datetime(close_date)`, res.ID, name, openDate, closeDate)
	if row.Err() != nil {
		return res, row.Err()
	}
	var (
		os string
		cs sql.NullString
	)
	if err = row.Scan(&res.Name, &os, &cs); err != nil {
		return res, err
	}
	if err = parseDatetime(os, &res.OpenDate); err != nil {
		return res, err
	}
	if cs.Valid {
		var t time.Time
		if err = parseDatetime(cs.String, &t); err != nil {
			return res, err
		}
		res.CloseDate = &t
	}
	return res, nil
}

// DeleteAccount deletes an account in the current version.
func DeleteAccount(ctx context.Context, db db, id int) (bool, error) {
	res, err := db.ExecContext(ctx, `
	  UPDATE accounts
	  SET version_to = (SELECT MAX(version) from versions)
	  WHERE accounts.account_id = ? 
	  AND version_to > (SELECT MAX(version) from versions)
	`, id)
	if err != nil {
		return false, err
	}
	rr, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rr == 1, nil
}
