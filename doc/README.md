# knut ‒ a plain text accounting tool

![Go](https://github.com/sboehler/knut/workflows/Go/badge.svg)

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It produces various reports based on simple accounting directives in a [text file](#file-format). knut is written in Go, and its primary use cases are personal finance and investing.

```text
{{ .Commands.BalanceIntro }}
```

## Table of contents

- [knut ‒ a plain text accounting tool](#knut--a-plain-text-accounting-tool)
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
    - [Transcode to beancount](#transcode-to-beancount)
  - [Editor support](#editor-support)
  - [File format](#file-format)
    - [Open and close](#open-and-close)
    - [Transactions](#transactions)
    - [Accruals (experimental)](#accruals-experimental)
    - [Balance assertions](#balance-assertions)
    - [Value directive](#value-directive)
    - [Prices](#prices)
    - [Include directives](#include-directives)

## Commands

```text
{{ .Commands.help }}
```

### Print a balance

knut has a powerful balance command, with various options to tune the result.

#### Basic balance

Without additional options, knut will print a balance with counts of the various
commodities per account.

```text
{{ .Commands.BalanceBasic }}
```

#### Monthly balance in a given commodity

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```text
{{ .Commands.BalanceMonthlyCHF }}
```

It balances in any currency:

```text
{{ .Commands.BalanceMonthlyUSD }}
```

#### Filter transactions by account or commodity

Use `--diff` to look into period differences. Use `--account` to filter for transactions affecting a single account, or `--commodity` to filter for transactions which affect a commodity. Both `--account` and `--commodity` take regular expressions, to select multiple matches.

```text
{{ .Commands.FilterAccount}}
```

```text
{{ .Commands.FilterCommodity}}
```

#### Collapse accounts

Use `-m` to map accounts matching a certain regex to a reduced number of segments. This can be used to completely hide an account (`-m0` - its positions will show up in the delta):

```text
{{ .Commands.Collapse}}
```

Alternatively, with a number > 0, subaccounts will be aggregated:

```text
{{ .Commands.Collapse1}}
```

### Fetch quotes

knut price sources are configured in yaml format:

```text
# doc/prices.yaml
{{ .PricesFile }}
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
{{ .Commands.HelpImport }}
```

### Transcode to beancount

While knut has advanced terminal-based visualization options, it lacks any web-based visualization tools. To allow the usage of the amazing tooling around the [beancount](http://furius.ca/beancount/) ecosystem, such as [fava](https://beancount.github.io/fava/), knut has a command to convert an entire journal into beancount's file format:

```text
knut transcode -c CHF doc/example.knut
```

This command should also allow beancount users to use knut's built-in importers.

## Editor support

There is an experimental [Visual Studio Code extension](https://github.com/sboehler/language-knut) which provides syntax highlighting, code folding and an outline view.

## File format

An accounting journal in knut is represented as a sequence of plain-text directives. The journal consists of a set of directives and comments. Directives are prices, account openings, transactions, value directives, balance assertions, and account closings. Lines starting with either `#` (comment) or `*` (org-mode title) are ignored. Files can include other files using an include directive. The order of the directives in the journal file is not important, they are always evaluated by date.

The following is an example for a knut journal:

```text
# doc/example.knut
{{ .ExampleFile }}
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

The transaction syntax deviates from similar tools like ledger or beancount for several reasons:

- It ensures that a transaction always balances, which is not guaranteed by formats where each booking references only one account.
- It creates unambigous flows between two accounts, which is helpful when analyzing the flows of money.
- The representation is more compact.

### Accruals (experimental)

Accruals are annotation placed on transactions to describe how the transaction's flows are to be broken up over time. Suppose you pay your yearly tax bill for 2020 on 24 March of that same year:

```text
2020-03-24 "2020 Taxes"
Assets:BankAccount Expenses:Taxes 12000 USD
```

This will heavily impact your net income in March due to the large cash outflow, while the taxes are actually owed for the entire year. Enter accruals:

```text
@accrue monthly 2020-01-01 2020-12-01 Assets:PrepaidTax
2020-03-24 "2020 Taxes"
Assets:BankAccount Expenses:Taxes 12000 USD
```

This annotation will replace the original transaction with an accrual, moving the money from expenses to a virtual asset account. In addition, the annotation will generate a series of small transactions which continuously move money from the virtual asset account to the expense account:

```text
# Accrual leg:
2020-03-24 "2020 Taxes"
Assets:BankAccount Assets:PrepaidTax 12000 USD

# Expense legs:
2020-01-31 "2020 Taxes"
Assets:PrepaidTax Expenses:Taxes 1000 USD

2020-02-29 "2020 Taxes"
Assets:PrepaidTax Expenses:Taxes 1000 USD

2020-03-31 "2020 Taxes"
Assets:PrepaidTax Expenses:Taxes 1000 USD

# ... etc, in total 12 transactions
```

knut will take care that the total impact remains the same. Also, amounts are properly split, without remainder.

```text
@accrue <once|daily|weekly|monthly|quarterly|yearly> <T0> <T1> <accrual account>
<transaction>
```

### Balance assertions

It is often helpful to check whether the balance at a date corresponds to an expected value, for example a value given by a bank account statement. A balance assertion in knut performs this check and reports an error if the check fails:

`YYYY-MM-DD balance <account> <amount> <commodity>`

### Value directive

Value directives can be used to declare a certain account balance at a specific date. When encountering a value directive during evaluation, knut will automatically generate a transaction wich makes sure that the balance matches the indicated value. The generated transaction always has exactly one booking, and the two accounts are the given account and a special Equity:Valuation account.

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
