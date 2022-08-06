package repo

import (
	"bytes"
	"encoding/gob"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/sboehler/knut/pkg/repo/schema"
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

func (db *DB) Read() *ReadTrx {
	return &ReadTrx{
		trx: db.bd.NewTransaction(false),
		db:  db,
	}
}

func (db *DB) Write() *WriteTrx {
	return &WriteTrx{
		ReadTrx{
			trx: db.bd.NewTransaction(true),
			db:  db,
		},
	}
}

type ReadTrx struct {
	trx *badger.Txn
	db  *DB
}

func (trx *ReadTrx) ReadAccount(id schema.AccountID) (*schema.Account, error) {
	k := fmt.Sprintf("tables/accounts/pk/%d", id)
	item, err := trx.trx.Get([]byte(k))
	if err != nil {
		return nil, err
	}
	var acc schema.Account
	err = item.Value(func(v []byte) error {
		err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(&acc)
		if err != nil {
			return fmt.Errorf("decode(%v): %w", v, err)
		}
		return nil
	})
	return &acc, err
}

type WriteTrx struct {
	ReadTrx
}

func (trx *WriteTrx) CreateAccount(acc *schema.Account) error {
	id, err := trx.db.GetID()
	if err != nil {
		return err
	}
	acc.ID = schema.AccountID(id)
	v, err := encode(acc)
	if err != nil {
		return err
	}
	k := fmt.Sprintf("tables/accounts/pk/%d", acc.ID)
	if err := trx.trx.Set([]byte(k), v); err != nil {
		return err
	}
	return nil
}

func encode[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(&v); err != nil {
		return nil, fmt.Errorf("enc.Encode(%v): %w", v, err)
	}
	return buf.Bytes(), nil
}
