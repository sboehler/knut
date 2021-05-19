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
    PRIMARY KEY(id, created_at)
);

CREATE TABLE prices_history (
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  target_commodity_id INTEGER NOT NULL REFERENCES commodities,
  price DOUBLE NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  PRIMARY KEY(date, commodity_id, target_commodity_id, created_at)
);

CREATE TABLE assertion_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE assertions_history (
  id INTEGER NOT NULL REFERENCES assertion_ids,
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  account_id INTEGER NOT NULL REFERENCES account_ids,
  amount TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  PRIMARY KEY(id, created_at)
);

CREATE TABLE transaction_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE transactions_history (
  id INTEGER NOT NULL REFERENCES transaction_ids,
  date TEXT NOT NULL,
  description TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  PRIMARY KEY(id, created_at)
);

CREATE TABLE bookings_history (
  id INTEGER NOT NULL,
  credit_account_id INTEGER NOT NULL REFERENCES account_ids,
  debit_account_id INTEGER NOT NULL REFERENCES account_ids,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  amount TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TEXT NOT NULL DEFAULT (DATETIME('2999-12-31')),
  FOREIGN KEY(id, created_at) REFERENCES transactions_history(id, created_at)
);
  
CREATE INDEX bookings_history_transaction_id_index on bookings_history(id, created_at);
CREATE INDEX bookings_history_credit_account_id_index on bookings_history(credit_account_id);
CREATE INDEX bookings_history_debit_account_id_index on bookings_history(debit_account_id);
CREATE INDEX bookings_history_commodity_id_index on bookings_history(commodity_id);
