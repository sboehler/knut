

# About knut

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It
produces various reports based on simple accounting directives in a [text file](#example-journal).
knut is written in Go, and its primary use cases are personal finance and
investing.

```
$ knut balance -v CHF -c0,(Income|Expenses|Equity) --monthly --from 2020-01-01 --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+
|    Account    | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |
|   BankAccount |      10000 |      11800 |      14127 |      14127 |      14127 |
|   Portfolio   |            |       1025 |        919 |        856 |        984 |
+---------------+------------+------------+------------+------------+------------+
| Total         |      10000 |      12825 |      15046 |      14983 |      15111 |
+---------------+------------+------------+------------+------------+------------+


```

# Table of contents
- [About knut](#about-knut)
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


# Commands

```
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

## Print a balance

knut has a powerful balance command, with various options to tune the result.

### Basic balance

Without additional options, knut will print a balance with counts of the various
commodities per account.

```
$ knut balance doc/example.knut
+---------------+------------+------------+
|    Account    | 2019-12-31 | 2020-11-20 |
+---------------+------------+------------+
| Assets        |            |            |
|   BankAccount |            |            |
|     CHF       |      10000 |      14127 |
|   Portfolio   |            |            |
|     AAPL      |            |         12 |
|     CHF       |            |         31 |
|     USD       |            |         97 |
+---------------+------------+------------+
| Equity        |            |            |
|   Equity      |            |            |
|     AAPL      |            |        -12 |
|     CHF       |     -10000 |      -9031 |
|     USD       |            |       -101 |
+---------------+------------+------------+
| Income        |            |            |
|   Salary      |            |            |
|     CHF       |            |     -10000 |
+---------------+------------+------------+
| Expenses      |            |            |
|   Fees        |            |            |
|     USD       |            |          4 |
|   Groceries   |            |            |
|     CHF       |            |        873 |
|   Rent        |            |            |
|     CHF       |            |       4000 |
+---------------+------------+------------+
| Total         |            |            |
|   AAPL        |            |            |
|   CHF         |            |            |
|   USD         |            |            |
+---------------+------------+------------+


```

### Monthly balance in CHF

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```
$ knut balance -v CHF --monthly --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+------------+
|    Account    | 2019-11-30 | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |            |
|   BankAccount |            |      10000 |      11800 |      14127 |      14127 |      14127 |
|   Portfolio   |            |            |       1025 |        919 |        856 |        984 |
+---------------+------------+------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |            |            |
|   Equity      |            |     -10000 |     -10003 |     -10003 |     -10003 |     -10003 |
|   Valuation   |            |            |        -26 |         80 |        143 |         15 |
+---------------+------------+------------+------------+------------+------------+------------+
| Income        |            |            |            |            |            |            |
|   Salary      |            |            |      -5000 |     -10000 |     -10000 |     -10000 |
+---------------+------------+------------+------------+------------+------------+------------+
| Expenses      |            |            |            |            |            |            |
|   Fees        |            |            |          4 |          4 |          4 |          4 |
|   Groceries   |            |            |        200 |        873 |        873 |        873 |
|   Rent        |            |            |       2000 |       4000 |       4000 |       4000 |
+---------------+------------+------------+------------+------------+------------+------------+
| Total         |            |            |            |            |            |            |
+---------------+------------+------------+------------+------------+------------+------------+


```

### Monthly income statement in CHF

TODO: option to reverse signs

```
$ knut balance -v CHF --monthly --to 2020-04-01 -c0,(Assets|Liabilities) --diff doc/example.knut
+-------------+------------+------------+------------+------------+------------+
|   Account   | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+-------------+------------+------------+------------+------------+------------+
| Equity      |            |            |            |            |            |
|   Equity    |     -10000 |         -3 |            |            |            |
|   Valuation |            |        -26 |        106 |         63 |       -127 |
+-------------+------------+------------+------------+------------+------------+
| Income      |            |            |            |            |            |
|   Salary    |            |      -5000 |      -5000 |            |            |
+-------------+------------+------------+------------+------------+------------+
| Expenses    |            |            |            |            |            |
|   Fees      |            |          4 |            |            |            |
|   Groceries |            |        200 |        673 |            |            |
|   Rent      |            |       2000 |       2000 |            |            |
+-------------+------------+------------+------------+------------+------------+
| Total       |     -10000 |      -2825 |      -2221 |         63 |       -127 |
+-------------+------------+------------+------------+------------+------------+


```

## Fetch quotes

knut price sources are configured in yaml format:
```
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

Once configure, prices can simply be updated:

```
knut fetch doc/prices.yaml
```

## Infer accounts

knut has a built-in Bayes engine to automatically assign accounts for new transactions. Simply use `TBD` as the account in a transaction and let knut decide how to replace it, based on previous entries. The bigger the journal, the more reliable this mechanism becomes.

```
knut infer -t doc/example.knut doc/example.knut
```

## Format the journal

knut can format a journal, such that accounts and numbers are aligned. Any comments and whitespace between directives are preserved.

```
knut format doc/example.knut
```

## Import transactions

knut has a few built-in importers for statements from Swiss banks:

```
$ knut import --help
Import financial account statements

Usage:
  knut import [command]

Available Commands:
  ch.cumulus            Import Cumulus credit card statements
  ch.postfinance        Import Postfinance CSV account statements
  ch.swisscard          Import Swisscard credit card statements
  ch.swissquote         Import Swissquote account reports
  us.interactivebrokers Import Interactive Brokers account reports

Flags:
  -h, --help   help for import

Use "knut import [command] --help" for more information about a command.

```

# Example journal

The journal consists of a set of directives and comments. Directives are prices, account
openings, transactions, balance assertions, and account closings. Comments start with either `#` or `*`.

Postings in transactions are written as 
`<debit account> <credit account> <amount> <commodity>`

```
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