CREATE TABLE versions (
    version_id INTEGER PRIMARY KEY,
    description TEXT,
    created_at TEXT
);

CREATE TABLE changelists (
    changelist_id INTEGER PRIMARY KEY,
    description TEXT,
    created_at TEXT
);

CREATE TABLE commodities(
    commodity_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE accounts(
    account_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    type INTEGER NOT NULL
);

CREATE TABLE directives(
    directive_id INTEGER PRIMARY KEY
);

CREATE TABLE directive_versions (
  directive_id INTEGER NOT NULL REFERENCES directives,
  directive_version_id INTEGER PRIMARY KEY,
  version_from INTEGER NOT NULL REFERENCES versions,
  version_to INTEGER REFERENCES versions,
  changelist_id INTEGER REFERENCES changelists
);

CREATE INDEX directive_versions_directive_id_index on directive_versions(directive_id);

CREATE TABLE prices (
  directive_versions_id INTEGER NOT NULL REFERENCES directive_versions,
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  target_commodity_id INTEGER NOT NULL REFERENCES commodities,
  price DOUBLE NOT NULL
);

CREATE INDEX prices_directive_versions_id_index on prices(directive_versions_id);
CREATE INDEX prices_commodity_id_index on prices(commodity_id);
CREATE INDEX prices_target_commodity_id_index on prices(target_commodity_id);

CREATE TABLE assertions (
  directive_versions_id INTEGER NOT NULL REFERENCES directive_versions,
  date TEXT NOT NULL,
  commodity_id INTEGER NOT NULL REFERENCES commodities,
  account_id INTEGER NOT NULL REFERENCES accounts,
  amount TEXT NOT NULL
);

CREATE INDEX assertions_directive_versions_id_index on assertions(directive_versions_id);
CREATE INDEX assertions_commodity_id_index on assertions(commodity_id);
CREATE INDEX assertions_account_id_index on assertions(account_id);


CREATE TABLE transactions (
  transaction_id INTEGER PRIMARY KEY,
  directive_versions_id INTEGER NOT NULL REFERENCES directive_versions,
  date TEXT NOT NULL,
  description TEXT NOT NULL
);

CREATE INDEX transactions_directive_versions_id_index on transactions(directive_versions_id);

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
