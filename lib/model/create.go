package model

import (
	"github.com/sboehler/knut/lib/syntax"
)

func createTransaction(reg *Registry, t *syntax.Transaction) *Transaction {
	return nil
}

func createAssertion(reg *Registry, a *syntax.Assertion) *Assertion {
	return nil
}

func createClose(reg *Registry, c *syntax.Close) *Close {
	return nil
}

func createOpen(reg *Registry, o *syntax.Open) *Open {
	return nil
}

// // Expand expands an accrual transaction.
// func (a Accrual) Expand(t *Transaction) []*Transaction {
// 	var result []*Transaction
// 	for _, p := range t.Postings {
// 		if p.Account.IsAL() {
// 			result = append(result, TransactionBuilder{
// 				Src:         t.Src,
// 				Date:        t.Date,
// 				Description: t.Description,
// 				Postings: PostingBuilder{
// 					Credit:    t.Accrual.Account,
// 					Debit:     p.Account,
// 					Commodity: p.Commodity,
// 					Amount:    p.Amount,
// 				}.Build(),
// 			}.Build())
// 		}
// 		if p.Account.IsIE() {
// 			partition := date.NewPartition(a.Period, a.Interval, 0)
// 			amount, rem := p.Amount.QuoRem(decimal.NewFromInt(int64(partition.Size())), 1)
// 			for i, dt := range partition.EndDates() {
// 				a := amount
// 				if i == 0 {
// 					a = a.Add(rem)
// 				}
// 				result = append(result, TransactionBuilder{
// 					Src:         t.Src,
// 					Date:        dt,
// 					Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, partition.Size()),
// 					Postings: PostingBuilder{
// 						Credit:    t.Accrual.Account,
// 						Debit:     p.Account,
// 						Commodity: p.Commodity,
// 						Amount:    a,
// 					}.Build(),
// 				}.Build())
// 			}
// 		}
// 	}
// 	return result
// }
