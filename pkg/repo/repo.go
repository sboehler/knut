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

func (db *DB) keyFor(v any) string {
	switch i := v.(type) {
	case *schema.Account:
		return fmt.Sprintf("tables/accounts/pk/%d", i.ID())
	}
	panic(fmt.Sprintf("invalid entity: %v", v))
}

type ReadTrx struct {
	trx *badger.Txn
	db  *DB
}

type WriteTrx struct {
	ReadTrx
}

type Entity[T any] interface {
	SetID(uint64)
	ID() uint64
	*T
}

func Create[T any, PT Entity[T]](trx *WriteTrx, e PT) error {
	id, err := trx.db.GetID()
	if err != nil {
		return err
	}
	e.SetID(id)
	v, err := encode(e)
	if err != nil {
		return err
	}
	k := trx.db.keyFor(e)
	if err := trx.trx.Set([]byte(k), v); err != nil {
		return err
	}
	return nil
}

func Read[T any, PT Entity[T]](trx *ReadTrx, id uint64) (PT, error) {
	acc := PT(new(T))
	acc.SetID(id)
	k := trx.db.keyFor(acc)
	item, err := trx.trx.Get([]byte(k))
	if err != nil {
		return nil, fmt.Errorf("trx.Get(%v): %w", k, err)
	}
	if err := decode(acc, item); err != nil {
		return nil, fmt.Errorf("decode(%v): %w", item, err)
	}
	return acc, nil

}

func encode[T any, PT *T](v PT) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("enc.Encode(%v): %w", v, err)
	}
	return buf.Bytes(), nil
}

func decode[T any](v T, item *badger.Item) error {
	return item.Value(func(bs []byte) error {
		err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(v)
		if err != nil {
			return fmt.Errorf("decode(%v): %w", bs, err)
		}
		return nil
	})
}
