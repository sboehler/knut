# knut - a plain text accounting tool

![Go](https://github.com/sboehler/knut/workflows/Go/badge.svg)

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It produces various reports based on simple accounting directives in a [text file](#file-format). knut is written in Go, and its primary use cases are personal finance and investing.

```text
$ knut balance -v CHF --months --from 2020-01-01 --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+
|    Account    | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |
|   BankAccount |     10,000 |     11,800 |     14,127 |     14,127 |     14,127 |
|   Portfolio   |            |      1,025 |        919 |        856 |        984 |
|               |            |            |            |            |            |
| Total         |     10,000 |     12,825 |     15,046 |     14,983 |     15,111 |
+---------------+------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |            |
|   Equity      |     10,000 |     10,003 |     10,003 |     10,003 |     10,003 |
|   Valuation   |            |         26 |        -80 |       -143 |        -15 |
|               |            |            |            |            |            |
| Income        |            |            |            |            |            |
|   Salary      |            |      5,000 |     10,000 |     10,000 |     10,000 |
|               |            |            |            |            |            |
| Expenses      |            |            |            |            |            |
|   Fees        |            |         -4 |         -4 |         -4 |         -4 |
|   Groceries   |            |       -200 |       -873 |       -873 |       -873 |
|   Rent        |            |     -2,000 |     -4,000 |     -4,000 |     -4,000 |
|               |            |            |            |            |            |
| Total         |     10,000 |     12,825 |     15,046 |     14,983 |     15,111 |
+---------------+------------+------------+------------+------------+------------+
| Delta         |            |            |            |            |            |
+---------------+------------+------------+------------+------------+------------+


```

## Table of contents

- [knut - a plain text accounting tool](#knut---a-plain-text-accounting-tool)
  - [Table of contents](#table-of-contents)
  - [Commands](#commands)
    - [Print a balance](#print-a-balance)
      - [Basic balance](#basic-balance)
      - [Monthly balance in a given commodity](#monthly-balance-in-a-given-commodity)
      - [Filter transactions by account or commodity](#filter-transactions-by-account-or-commodity)
      - [Collapse accounts](#collapse-accounts)
    - [Fetch quotes](#fetch-quotes)
    - [Infer accounts](#infer-accounts)
    - [Format the journal](#format-the-journal)
    - [Import transactions](#import-transactions)
  - [File format](#file-format)
    - [Open and close](#open-and-close)
    - [Transactions](#transactions)
    - [Balance assertions](#balance-assertions)
    - [Value directive](#value-directive)
    - [Prices](#prices)
    - [Include directives](#include-directives)

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
|               |            |            |
| Total         |            |            |
|   AAPL        |            |         12 |
|   CHF         |     10,000 |     14,158 |
|   USD         |            |         97 |
+---------------+------------+------------+
| Equity        |            |            |
|   Equity      |            |            |
|     AAPL      |            |         12 |
|     CHF       |     10,000 |      9,031 |
|     USD       |            |        101 |
|               |            |            |
| Income        |            |            |
|   Salary      |            |            |
|     CHF       |            |     10,000 |
|               |            |            |
| Expenses      |            |            |
|   Fees        |            |            |
|     USD       |            |         -4 |
|   Groceries   |            |            |
|     CHF       |            |       -873 |
|   Rent        |            |            |
|     CHF       |            |     -4,000 |
|               |            |            |
| Total         |            |            |
|   AAPL        |            |         12 |
|   CHF         |     10,000 |     14,158 |
|   USD         |            |         97 |
+---------------+------------+------------+
| Delta         |            |            |
|   AAPL        |            |            |
|   CHF         |            |            |
|   USD         |            |            |
+---------------+------------+------------+


```

#### Monthly balance in a given commodity

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```text
$ knut balance -v CHF --months --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+------------+
|    Account    | 2019-11-30 | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |            |
|   BankAccount |            |     10,000 |     11,800 |     14,127 |     14,127 |     14,127 |
|   Portfolio   |            |            |      1,025 |        919 |        856 |        984 |
|               |            |            |            |            |            |            |
| Total         |            |     10,000 |     12,825 |     15,046 |     14,983 |     15,111 |
+---------------+------------+------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |            |            |
|   Equity      |            |     10,000 |     10,003 |     10,003 |     10,003 |     10,003 |
|   Valuation   |            |            |         26 |        -80 |       -143 |        -15 |
|               |            |            |            |            |            |            |
| Income        |            |            |            |            |            |            |
|   Salary      |            |            |      5,000 |     10,000 |     10,000 |     10,000 |
|               |            |            |            |            |            |            |
| Expenses      |            |            |            |            |            |            |
|   Fees        |            |            |         -4 |         -4 |         -4 |         -4 |
|   Groceries   |            |            |       -200 |       -873 |       -873 |       -873 |
|   Rent        |            |            |     -2,000 |     -4,000 |     -4,000 |     -4,000 |
|               |            |            |            |            |            |            |
| Total         |            |     10,000 |     12,825 |     15,046 |     14,983 |     15,111 |
+---------------+------------+------------+------------+------------+------------+------------+
| Delta         |            |            |            |            |            |            |
+---------------+------------+------------+------------+------------+------------+------------+


```

It balances in any currency:

```text
$ knut balance -v USD --months --to 2020-04-01 doc/example.knut
+---------------+------------+------------+------------+------------+------------+------------+
|    Account    | 2019-11-30 | 2019-12-31 | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |            |            |
|   BankAccount |            |     10,324 |     12,172 |     14,583 |     14,716 |     14,506 |
|   Portfolio   |            |            |      1,058 |        949 |        892 |      1,010 |
|               |            |            |            |            |            |            |
| Total         |            |     10,324 |     13,230 |     15,532 |     15,608 |     15,516 |
+---------------+------------+------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |            |            |
|   Equity      |            |     10,324 |     10,327 |     10,327 |     10,327 |     10,327 |
|   Valuation   |            |            |         24 |        -25 |         51 |        -41 |
|               |            |            |            |            |            |            |
| Income        |            |            |            |            |            |            |
|   Salary      |            |            |      5,157 |     10,260 |     10,260 |     10,260 |
|               |            |            |            |            |            |            |
| Expenses      |            |            |            |            |            |            |
|   Fees        |            |            |         -4 |         -4 |         -4 |         -4 |
|   Groceries   |            |            |       -207 |       -896 |       -896 |       -896 |
|   Rent        |            |            |     -2,067 |     -4,130 |     -4,130 |     -4,130 |
|               |            |            |            |            |            |            |
| Total         |            |     10,324 |     13,230 |     15,532 |     15,608 |     15,516 |
+---------------+------------+------------+------------+------------+------------+------------+
| Delta         |            |            |            |            |            |            |
+---------------+------------+------------+------------+------------+------------+------------+


```

#### Filter transactions by account or commodity

Use `--diff` to look into period differences. Use `--account` to filter for transactions affecting a single account, or `--commodity` to filter for transactions which affect a commodity. Both `--account` and `--commodity` take regular expressions, to select multiple matches.

```text
$ knut balance -v CHF --months --from 2020-01-01 --to 2020-04-01 --diff --account Portfolio doc/example.knut
+---------------+------------+------------+------------+------------+
|    Account    | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |
|   BankAccount |     -1,000 |            |            |            |
|   Portfolio   |      1,025 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Total         |         25 |       -106 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |
|   Equity      |          3 |            |            |            |
|   Valuation   |         26 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Expenses      |            |            |            |            |
|   Fees        |         -4 |            |            |            |
|               |            |            |            |            |
| Total         |         25 |       -106 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Delta         |            |            |            |            |
+---------------+------------+------------+------------+------------+


```

```text
$ knut balance -v CHF --months --from 2020-01-01 --to 2020-04-01 --diff --commodity AAPL doc/example.knut
+-------------+------------+------------+------------+------------+
|   Account   | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+-------------+------------+------------+------------+------------+
| Assets      |            |            |            |            |
|   Portfolio |        900 |       -106 |        -62 |        126 |
|             |            |            |            |            |
| Total       |        900 |       -106 |        -62 |        126 |
+-------------+------------+------------+------------+------------+
| Equity      |            |            |            |            |
|   Equity    |        874 |            |            |            |
|   Valuation |         26 |       -106 |        -62 |        126 |
|             |            |            |            |            |
| Total       |        900 |       -106 |        -62 |        126 |
+-------------+------------+------------+------------+------------+
| Delta       |            |            |            |            |
+-------------+------------+------------+------------+------------+


```

#### Collapse accounts

Use `-c` to collapse accounts matching a certain regex. This can be used to completely hide an account (`-c0` - its positions will show up in the delta):

```text
$ knut balance -v CHF --months --from 2020-01-01 --to 2020-04-01 --diff -c0,(Income|Expenses) doc/example.knut
+---------------+------------+------------+------------+------------+
|    Account    | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |
|   BankAccount |      1,800 |      2,327 |            |            |
|   Portfolio   |      1,025 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Total         |      2,825 |      2,221 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Equity        |            |            |            |            |
|   Equity      |          3 |            |            |            |
|   Valuation   |         26 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Total         |         29 |       -106 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Delta         |      2,796 |      2,327 |            |            |
+---------------+------------+------------+------------+------------+


```

Alternatively, with a number > 0, subaccounts will be aggregated:

```text
$ knut balance -v CHF --months --from 2020-01-01 --to 2020-04-01 --diff -c1,(Income|Expenses|Equity) doc/example.knut
+---------------+------------+------------+------------+------------+
|    Account    | 2020-01-31 | 2020-02-29 | 2020-03-31 | 2020-04-30 |
+---------------+------------+------------+------------+------------+
| Assets        |            |            |            |            |
|   BankAccount |      1,800 |      2,327 |            |            |
|   Portfolio   |      1,025 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Total         |      2,825 |      2,221 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Equity        |         29 |       -106 |        -63 |        127 |
|               |            |            |            |            |
| Income        |      5,000 |      5,000 |            |            |
|               |            |            |            |            |
| Expenses      |     -2,204 |     -2,673 |            |            |
|               |            |            |            |            |
| Total         |      2,825 |      2,221 |        -63 |        127 |
+---------------+------------+------------+------------+------------+
| Delta         |            |            |            |            |
+---------------+------------+------------+------------+------------+


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

Once configured, prices can be updated with one command:

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

## File format

An accounting journal in knut is represented as a sequence of plain-text directives. The journal consists of a set of directives and comments. Directives are prices, account openings, transactions, value directives, balance assertions, and account closings. Lines starting with either `#` (comment) or `*` (org-mode title) are ignored. Files can include other files using an include directive. The order of the directives in the journal file is not important, they are always evaluated by date.

The following is an example for a knut journal:

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

### Open and close

An account consists of a sequence of segments, separated by ':'. The first segment must be one of Assets, Liabilities, Equity, Income, Expenses or TBD. Before an account can be used in a transaction, for example, it must be opened using an open directive:

`YYYY-MM-DD open <account name>`

Once an account is not needed anymore, it can be closed, to prevent further bookings. An account can only be closed if its balance is zero at the closing time.

`YYYY-MM-DD close <account name>`

### Transactions

A transaction describes the flow of money between multiple accounts. Transaction always balance by design in knut.

```text
YYYY-MM-DD "<description>"
<credit account> <debit account> <amount> <commodity>
<credit account> <debit account> <amount> <commodity>
...
```

A transaction starts with a date, followed by a description withing double quotes on the same line. It must have one or more bookings on the lines immediately following. Every booking references two accounts, a credit account (first) and a debit account (second). The amount is usually a positive numbers, and the semantics is that money "flows from left to right".

The transaction syntaxt deviates from similar tools like ledger or beancount for several reasons:

- It ensures that a transaction always balances, which is not guaranteed by formats where each booking references only one account.
- It creates unambigous flows between two accounts, which is helpful when analyzing the flows of money.
- The representation is more compact.

### Balance assertions

It is often helpful to check whether the balance at a date corresponds to an expected value, for example a value given by a bank account statement. A balance assertion in knut performs this check and reports an error if the check fails:

`YYYY-MM-DD balance <account> <amount> <commodity>`

### Value directive

Value directives can be used to declare a certain account balance at a specific date. When encountering a value directive during evaluation, beans will automatically generate a transaction wich makes sure that the balance matches the indicated value. The generated transaction always has exactly one booking, and the two accounts are the given account and a special Equity:Valuation account.

`YYYY-MM-DD value <account> <amount> <commodity>`

Value directives are handy in particular for modeling investment portfolios, where it is too much work to model every individual trade, for example in an automated trading system. In such a situation, declare inflows and outflows of the investment as usual, and provide value directives for any day the value of the investment can be established (ideally daily). knut will automatically generate transaction representing the value changes of the investment, after considering any given bookings affecting the account.

### Prices

knut has a power valuation engine, which can be used to create balance sheets in any currency or security which has pricing information. Prices are declared using price directives:

`YYYY-MM-DD price <commodity> <price> <target_commodity>`

For example, `2020-10-03 price AAPL 45 USD` declares that AAPL cost 45 USD on 2020-10-03 (you wish...). knut is smart enough to derive indirect prices. For example, knut can print a balance with an AAPL position in CHF if a price for USD in CHF and a price for AAPL in USD exists. Prices are automatically inverted, as needed. knut will always use the latest available price for every given day. If a valuation is requried for a date before the first price is given, an error is reported.

### Include directives

Income directives can be used to split a journal across a set of files. The given path is interpreted relative to the location of the file where the include directive appears.

`include "<relative path>"`

It is entirely a matter of preference whether to use large files or a set of smaller files. knut ignores lines starting with '\*', so those with a [powerful editor](http://www.emacs.org) can use org-mode to fold sections of a file, making it easy to manage files with tens of thousands of lines.
