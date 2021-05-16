CREATE TABLE versions (
    id INTEGER PRIMARY KEY,
    description TEXT,
    created_at TEXT
);

CREATE TABLE commodities(
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE account_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE accounts(
    account_id INTEGER NOT NULL REFERENCES account_ids,
    name TEXT NOT NULL,
    open_date TEXT NOT NULL,
    close_date TEXT,
    version_from INTEGER NOT NULL REFERENCES versions,
    version_to INTEGER NOT NULL REFERENCES versions,
    PRIMARY KEY(account_id, version_from, version_to)
);

CREATE TABLE prices (
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  target_commodity_id INTEGER NOT NULL REFERENCES commodities,
  price DOUBLE NOT NULL,
  version_from INTEGER NOT NULL REFERENCES versions,
  version_to INTEGER NOT NULL REFERENCES versions,
  PRIMARY KEY(date, commodity_id, target_commodity_id, version_from, version_to)
);

CREATE TABLE assertion_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE assertions (
  assertion_id INTEGER NOT NULL REFERENCES assertion_ids,
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  account_id INTEGER NOT NULL REFERENCES accounts,
  amount TEXT NOT NULL,
  version_from INTEGER NOT NULL REFERENCES versions,
  version_to INTEGER NOT NULL REFERENCES versions,
  PRIMARY KEY(assertion_id, version_from, version_to)
);

CREATE TABLE transaction_ids (
  id INTEGER PRIMARY KEY
);

CREATE TABLE transactions (
  id INTEGER PRIMARY KEY,
  transaction_id INTEGER NOT NULL REFERENCES transaction_ids,
  date TEXT NOT NULL,
  description TEXT NOT NULL,
  version_from INTEGER NOT NULL REFERENCES versions,
  version_to INTEGER NOT NULL REFERENCES versions,
  UNIQUE(transaction_id, version_from, version_to)
);

CREATE TABLE bookings (
  transaction_id INTEGER NOT NULL REFERENCES transactions,
  credit_account_id INTEGER NOT NULL REFERENCES accounts,
  debit_account_id INTEGER NOT NULL REFERENCES accounts,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  amount TEXT NOT NULL
);
  
CREATE INDEX bookings_transaction_id_index on bookings(transaction_id);
CREATE INDEX bookings_credit_account_id_index on bookings(credit_account_id);
CREATE INDEX bookings_debit_account_id_index on bookings(debit_account_id);
CREATE INDEX bookings_commodity_id_index on bookings(commodity_id);
