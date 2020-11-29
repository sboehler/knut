# knut - a plain text accounting tool

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It
produces various reports based on simple accounting directives in a [text file](#example-journal).
knut is written in Go, and its primary use cases are personal finance and
investing.

```text
{{ .Commands.BalanceIntro }}
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

#### Monthly balance in CHF

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```text
{{ .Commands.BalanceMonthlyCHF }}
```

#### Monthly income statement in CHF

TODO: option to reverse signs

```text
{{ .Commands.IncomeMonthlyCHF}}
```

### Fetch quotes

knut price sources are configured in yaml format:

```text
# doc/prices.yaml
{{ .PricesFile }}
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
{{ .Commands.HelpImport }}
```

## Example journal

The journal consists of a set of directives and comments. Directives are prices, account
openings, transactions, balance assertions, and account closings. Comments start with either `#` or `*`.

Postings in transactions are written as `<credit account> <debit account> <amount> <commodity>`. One can think of money flowing from the left.

```text
# doc/example.knut
{{ .ExampleFile }}
```
