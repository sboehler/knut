CREATE TABLE commodities(
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE account_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE accounts_history(
    id INTEGER NOT NULL REFERENCES account_ids,
    name TEXT NOT NULL,
    open_date TEXT NOT NULL,
    close_date TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
    PRIMARY KEY(id, created_at, deleted_at)
);

CREATE TABLE prices (
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  target_commodity_id INTEGER NOT NULL REFERENCES commodities,
  price DOUBLE NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  PRIMARY KEY(date, commodity_id, target_commodity_id, created_at, deleted_at)
);

CREATE TABLE assertion_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE assertions (
  id INTEGER NOT NULL REFERENCES assertion_ids,
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  account_id INTEGER NOT NULL REFERENCES account_ids,
  amount TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  PRIMARY KEY(id, created_at, deleted_at)
);

CREATE TABLE transaction_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE transactions (
  id INTEGER NOT NULL REFERENCES transaction_ids,
  transaction_id INTEGER PRIMARY KEY,
  date TEXT NOT NULL,
  description TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  UNIQUE(id, created_at, deleted_at)
);

CREATE TABLE bookings (
  transaction_id INTEGER NOT NULL REFERENCES transactions,
  credit_account_id INTEGER NOT NULL REFERENCES account_ids,
  debit_account_id INTEGER NOT NULL REFERENCES account_ids,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  amount TEXT NOT NULL
);
  
CREATE INDEX bookings_transaction_id_index on bookings(transaction_id);
CREATE INDEX bookings_credit_account_id_index on bookings(credit_account_id);
CREATE INDEX bookings_debit_account_id_index on bookings(debit_account_id);
CREATE INDEX bookings_commodity_id_index on bookings(commodity_id);
