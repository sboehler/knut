package db

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	"github.com/dgraph-io/badger/v3"
)

const (
	namespaceTable byte = 0x1
	namespaceIndex byte = 0x2
)

type Entity[T any] interface {
	SetID(uint64)
	ID() uint64
	*T
}

type Table[T any, PT Entity[T]] struct {
	Name    string
	TableID byte
}

func (t Table[T, PT]) Create(trx *WriteTrx, e PT) error {
	id, err := trx.db.GetID()
	if err != nil {
		return err
	}
	e.SetID(id)
	return t.Update(trx, e)
}

func (t Table[T, PT]) Read(trx *ReadTrx, id uint64) (PT, error) {
	e := PT(new(T))
	e.SetID(id)
	k := t.Key(e)
	item, err := trx.trx.Get(k)
	if err != nil {
		return nil, fmt.Errorf("trx.Get(%v): %w", k, err)
	}
	if err := t.decode(e, item); err != nil {
		return nil, fmt.Errorf("decode(%v): %w", item, err)
	}
	return e, nil
}

func (t Table[T, PT]) Update(trx *WriteTrx, e PT) error {
	k := t.Key(e)
	v, err := t.encode(e)
	if err != nil {
		return err
	}
	if err := trx.trx.Set(k, v); err != nil {
		return fmt.Errorf("trx.Set(%v, %v): %w", k, v, err)
	}
	return nil
}

func (t Table[T, PT]) Delete(trx *WriteTrx, id uint64) error {
	e := PT(new(T))
	e.SetID(id)
	k := t.Key(e)
	if err := trx.trx.Delete(k); err != nil {
		return fmt.Errorf("trx.Delete(%v): %w", k, err)
	}
	return nil
}

func (t Table[T, PT]) encode(v PT) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("enc.Encode(%v): %w", v, err)
	}
	return buf.Bytes(), nil
}

func (t Table[T, PT]) decode(v PT, item *badger.Item) error {
	return item.Value(func(bs []byte) error {
		err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(v)
		if err != nil {
			return fmt.Errorf("decode(%v): %w", bs, err)
		}
		return nil
	})
}

func (t Table[T, PT]) Key(e PT) []byte {
	buf := bytes.NewBuffer([]byte{namespaceTable})
	buf.Write([]byte{byte(t.TableID), 0x1}) // accounts table, primary key
	binary.Write(buf, binary.BigEndian, e.ID())
	return buf.Bytes()
}
