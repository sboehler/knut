# knut - a plain text accounting tool

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It
produces various reports based on simple accounting directives in a [text file](#example-journal).
knut is written in Go, and its primary use cases are personal finance and
investing.

```text
$ knut balance -v CHF -c0,(Income|Expenses|Equity) --monthly --from 2020-01-01 --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+
|    Account    | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |
|   BankAccount |     10,000 |     11,800 |     14,127 |     14,127 |     14,127 |
|   Portfolio   |            |      1,025 |        919 |        856 |        984 |
+---------------+------------+------------+------------+------------+------------+
| Total         |     10,000 |     12,825 |     15,046 |     14,983 |     15,111 |
+---------------+------------+------------+------------+------------+------------+


```

## Table of contents

- [knut - a plain text accounting tool](#knut---a-plain-text-accounting-tool)
  - [Table of contents](#table-of-contents)
  - [Commands](#commands)
    - [Print a balance](#print-a-balance)
      - [Basic balance](#basic-balance)
      - [Monthly balance in CHF](#monthly-balance-in-chf)
      - [Monthly income statement in CHF](#monthly-income-statement-in-chf)
    - [Fetch quotes](#fetch-quotes)
    - [Infer accounts](#infer-accounts)
    - [Format the journal](#format-the-journal)
    - [Import transactions](#import-transactions)
  - [Example journal](#example-journal)

## Commands

```text
$ knut --help
knut is a plain text accounting tool for tracking personal finances and investments.

Usage:
  knut [command]

Available Commands:
  balance     create a balance sheet
  fetch       Fetch quotes from Yahoo! Finance
  format      Format the given journal
  help        Help about any command
  import      Import financial account statements
  infer       Auto-assign accounts in a journal

Flags:
  -h, --help   help for knut

Use "knut [command] --help" for more information about a command.

```

### Print a balance

knut has a powerful balance command, with various options to tune the result.

#### Basic balance

Without additional options, knut will print a balance with counts of the various
commodities per account.

```text
$ knut balance doc/example.knut
+---------------+------------+------------+
|    Account    | 2019-12-31 | 2020-11-20 |
+---------------+------------+------------+
| Assets        |            |            |
|   BankAccount |            |            |
|     CHF       |     10,000 |     14,127 |
|   Portfolio   |            |            |
|     AAPL      |            |         12 |
|     CHF       |            |         31 |
|     USD       |            |         97 |
+---------------+------------+------------+
| Equity        |            |            |
|   Equity      |            |            |
|     AAPL      |            |        -12 |
|     CHF       |    -10,000 |     -9,031 |
|     USD       |            |       -101 |
+---------------+------------+------------+
| Income        |            |            |
|   Salary      |            |            |
|     CHF       |            |    -10,000 |
+---------------+------------+------------+
| Expenses      |            |            |
|   Fees        |            |            |
|     USD       |            |          4 |
|   Groceries   |            |            |
|     CHF       |            |        873 |
|   Rent        |            |            |
|     CHF       |            |      4,000 |
+---------------+------------+------------+
| Total         |            |            |
|   AAPL        |            |            |
|   CHF         |            |            |
|   USD         |            |            |
+---------------+------------+------------+


```

#### Monthly balance in CHF

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```text
$ knut balance -v CHF --monthly --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+------------+
|    Account    | 2019-11-30 | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |            |
|   BankAccount |            |     10,000 |     11,800 |     14,127 |     14,127 |     14,127 |
|   Portfolio   |            |            |      1,025 |        919 |        856 |        984 |
+---------------+------------+------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |            |            |
|   Equity      |            |    -10,000 |    -10,003 |    -10,003 |    -10,003 |    -10,003 |
|   Valuation   |            |            |        -26 |         80 |        143 |         15 |
+---------------+------------+------------+------------+------------+------------+------------+
| Income        |            |            |            |            |            |            |
|   Salary      |            |            |     -5,000 |    -10,000 |    -10,000 |    -10,000 |
+---------------+------------+------------+------------+------------+------------+------------+
| Expenses      |            |            |            |            |            |            |
|   Fees        |            |            |          4 |          4 |          4 |          4 |
|   Groceries   |            |            |        200 |        873 |        873 |        873 |
|   Rent        |            |            |      2,000 |      4,000 |      4,000 |      4,000 |
+---------------+------------+------------+------------+------------+------------+------------+
| Total         |            |            |            |            |            |            |
+---------------+------------+------------+------------+------------+------------+------------+


```

#### Monthly income statement in CHF

TODO: option to reverse signs

```text
$ knut balance -v CHF --monthly --to 2020-04-01 -c0,(Assets|Liabilities) --diff doc/example.knut
+-------------+------------+------------+------------+------------+------------+
|   Account   | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+-------------+------------+------------+------------+------------+------------+
| Equity      |            |            |            |            |            |
|   Equity    |    -10,000 |         -3 |            |            |            |
|   Valuation |            |        -26 |        106 |         63 |       -127 |
+-------------+------------+------------+------------+------------+------------+
| Income      |            |            |            |            |            |
|   Salary    |            |     -5,000 |     -5,000 |            |            |
+-------------+------------+------------+------------+------------+------------+
| Expenses    |            |            |            |            |            |
|   Fees      |            |          4 |            |            |            |
|   Groceries |            |        200 |        673 |            |            |
|   Rent      |            |      2,000 |      2,000 |            |            |
+-------------+------------+------------+------------+------------+------------+
| Total       |    -10,000 |     -2,825 |     -2,221 |         63 |       -127 |
+-------------+------------+------------+------------+------------+------------+


```

### Fetch quotes

knut price sources are configured in yaml format:

```text
# doc/prices.yaml
- commodity: "USD"
  target_commodity: "CHF"
  file: "USD.prices"
  symbol: "USDCHF=X"
- commodity: "AAPL"
  target_commodity: "USD"
  file: "AAPL.prices"
  symbol: "AAPL"

```

Once configured, prices can simply be updated:

```text
knut fetch doc/prices.yaml
```

### Infer accounts

knut has a built-in Bayes engine to automatically assign accounts for new transactions. Simply use `TBD` as the account in a transaction and let knut decide how to replace it, based on previous entries. The bigger the journal, the more reliable this mechanism becomes.

```text
knut infer -t doc/example.knut doc/example.knut
```

### Format the journal

knut can format a journal, such that accounts and numbers are aligned. Any comments and whitespace between directives are preserved.

```text
knut format doc/example.knut
```

### Import transactions

knut has a few built-in importers for statements from Swiss banks:

```text
$ knut import --help
Import financial account statements

Usage:
  knut import [command]

Available Commands:
  ch.cumulus            Import Cumulus credit card statements
  ch.postfinance        Import Postfinance CSV account statements
  ch.swisscard          Import Swisscard credit card statements
  ch.swissquote         Import Swissquote account reports
  ch.viac               Import VIAC values from JSON files
  revolut               Import Revolut CSV account statements
  us.interactivebrokers Import Interactive Brokers account reports

Flags:
  -h, --help   help for import

Use "knut import [command] --help" for more information about a command.

```

## Example journal

The journal consists of a set of directives and comments. Directives are prices, account
openings, transactions, balance assertions, and account closings. Comments start with either `#` or `*`.

Postings in transactions are written as `<credit account> <debit account> <amount> <commodity>`. One can think of money flowing from the left.

```text
# doc/example.knut
include "USD.prices"
include "AAPL.prices"

* Open Accounts

2019-12-31 open Equity:Equity
2019-12-31 open Assets:BankAccount
2019-12-31 open Assets:Portfolio

2019-12-31 open Expenses:Groceries
2019-12-31 open Expenses:Fees
2019-12-31 open Expenses:Rent

2019-12-31 open Income:Salary
2019-12-31 open Income:Dividends

* Opening Balances

2019-12-31 "Opening balance"
Equity:Equity Assets:BankAccount 10000 CHF

* 2020-01

2020-01-25 "Salary January 2020"
Income:Salary Assets:BankAccount 5000 CHF

2020-01-02 "Rent January"
Assets:BankAccount Expenses:Rent 2000 CHF

2020-01-15 "Groceries"
Assets:BankAccount Expenses:Groceries 200 CHF

2020-01-05 "Transfer to portfolio"
Assets:BankAccount Assets:Portfolio 1000 CHF

2020-01-06 "Currency exchange"
Equity:Equity Assets:Portfolio 1001 USD
Assets:Portfolio Equity:Equity 969 CHF

2020-01-06 "Buy 3 AAPL shares"
Equity:Equity Assets:Portfolio 12 AAPL
Assets:Portfolio Equity:Equity 900 USD
Assets:Portfolio Expenses:Fees 4 USD

* 2020-02

2020-02-25 "Salary January 2020"
Income:Salary Assets:BankAccount 5000 CHF

2020-02-02 "Rent January"
Assets:BankAccount Expenses:Rent 2000 CHF

2020-02-05 "Groceries"
Assets:BankAccount Expenses:Groceries 250 CHF

2020-02-25 "Groceries"
Assets:BankAccount Expenses:Groceries 423 CHF

```
