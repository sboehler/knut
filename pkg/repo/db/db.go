package db

import (
	"github.com/dgraph-io/badger/v3"
)

type DB struct {
	bd *badger.DB
}

const (
	IDSequence = "id_sequence"
)

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

func (db *DB) GetID() (uint64, error) {
	s, err := db.bd.GetSequence([]byte(IDSequence), 1)
	if err != nil {
		return 0, err
	}
	defer s.Release()
	id, err := s.Next()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DB) Read(f func(*ReadTrx) error) error {
	return db.bd.View(func(txn *badger.Txn) error {
		trx := &ReadTrx{
			trx: txn,
			db:  db,
		}
		return f(trx)
	})
}

func (db *DB) Write(f func(*WriteTrx) error) error {
	return db.bd.Update(func(txn *badger.Txn) error {
		trx := &WriteTrx{
			ReadTrx{
				trx: txn,
				db:  db,
			}}
		return f(trx)
	})
}

type ReadTrx struct {
	trx *badger.Txn
	db  *DB
}

type WriteTrx struct {
	ReadTrx
}
