package model

import "github.com/sboehler/knut/lib/common/compare"

func CompareAccounts(a1, a2 *Account) compare.Order {
	o := compare.Ordered(a1.accountType, a2.accountType)
	if o != compare.Equal {
		return o
	}
	return compare.Ordered(a1.name, a2.name)
}

// Less defines an order on transactions.
func CompareTransactions(t *Transaction, t2 *Transaction) compare.Order {
	if o := compare.Time(t.Date, t2.Date); o != compare.Equal {
		return o
	}
	if o := compare.Ordered(t.Description, t2.Description); o != compare.Equal {
		return o
	}
	for i := 0; i < len(t.Postings) && i < len(t2.Postings); i++ {
		if o := ComparePostings(t.Postings[i], t2.Postings[i]); o != compare.Equal {
			return o
		}
	}
	return compare.Ordered(len(t.Postings), len(t2.Postings))
}
