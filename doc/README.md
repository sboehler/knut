

# About knut

> there are 29 knuts in a sickle

knut is a plain-text, double-entry accounting tool for the command line. It
produces various reports based on simple accounting directives in a [text file](#example-journal).
knut is written in Go, and its primary use cases are personal finance and
investing.

```
{{ .Commands.BalanceIntro }}
```

# Table of contents
- [About knut](#about-knut)
- [Table of contents](#table-of-contents)
- [Commands](#commands)
  - [Printing a balance](#printing-a-balance)
    - [Basic balance](#basic-balance)
    - [Monthly balance in CHF](#monthly-balance-in-chf)
    - [Monthly income statement in CHF](#monthly-income-statement-in-chf)
  - [Fetching prices](#fetching-prices)
  - [Infer accounts](#infer-accounts)
  - [Format the journal](#format-the-journal)
  - [Import transactions](#import-transactions)
- [Example journal](#example-journal)


# Commands

```
{{ .Commands.help }}
```

## Printing a balance

knut has a powerful balance command, with various options to tune the result.

### Basic balance

Without additional options, knut will print a balance with counts of the various
commodities per account.

```
{{ .Commands.BalanceBasic }}
```

### Monthly balance in CHF

If prices are available, knut can valuate the balance in any of the available commodities. And the result is guaranteed to balance:

```
{{ .Commands.BalanceMonthlyCHF }}
```

### Monthly income statement in CHF

TODO: option to reverse signs

```
{{ .Commands.IncomeMonthlyCHF}}
```

## Fetching prices

knut price sources are configured in yaml format:
```
# doc/prices.yaml
{{ .PricesFile }}
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
{{ .Commands.HelpImport }}
```

# Example journal

The journal consists of a set of directives and comments. Directives are prices, account
openings, transactions, balance assertions, and account closings. Comments start with either `#` or `*`.

Postings in transactions are written as 
`<debit account> <credit account> <amount> <commodity>`

```
# doc/example.knut
{{ .ExampleFile }}
```