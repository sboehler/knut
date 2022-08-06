package repo

import (
	badger "github.com/dgraph-io/badger/v3"
)

type DB struct {
	bd *badger.DB
}

func Open(path string) (*DB, error) {
	b, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}
	return &DB{
		bd: b,
	}, nil
}

func (db *DB) Close() error {
	return db.bd.Close()
}
