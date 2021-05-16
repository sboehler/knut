package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/sboehler/knut/lib/server/model"
)

// CreateVersion creates a new version.
func CreateVersion(ctx context.Context, db db, desc string) (model.Version, error) {
	var (
		row *sql.Row
		res = model.Version{
			Description: desc,
		}
		err error
	)
	if row = db.QueryRowContext(ctx,
		`INSERT INTO versions(version, description)
		SELECT (SELECT COALESCE(MAX(version), 0) +1 FROM versions), ?
		RETURNING version, created_at`, desc); row.Err() != nil {
		return res, row.Err()
	}
	var s string
	if err := row.Scan(&res.Version, &s); err != nil {
		return res, err
	}
	if err = parseDatetime(s, &res.CreatedAt); err != nil {
		return res, err
	}
	return res, nil
}

// ListVersions lists all versions.
func ListVersions(ctx context.Context, db db) ([]model.Version, error) {
	var res []model.Version
	rows, err := db.QueryContext(ctx, "SELECT version, description, datetime(created_at) FROM versions ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			c model.Version
			s string
		)
		if err := rows.Scan(&c.Version, &c.Description, &s); err != nil {
			return nil, err
		}
		if err = parseDatetime(s, &c.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	if rows.Err() != nil && rows.Err() != sql.ErrNoRows {
		return nil, rows.Err()
	}
	return res, nil
}

// DeleteVersion deletes the latest version.
func DeleteVersion(ctx context.Context, db db) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM versions 
		WHERE version = (SELECT MAX(version) FROM versions)`); err != nil {
		return err
	}
	return nil
}

func parseDatetime(s string, dest *time.Time) error {
	dt, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return err
	}
	*dest = dt
	return nil
}
