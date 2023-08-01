package transaction

import (
	"time"

	"github.com/sboehler/knut/lib/common/compare"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/syntax"
)

// Transaction represents a transaction.
type Transaction struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*posting.Posting
	Targets     []*commodity.Commodity
}

// Less defines an order on transactions.
func Compare(t *Transaction, t2 *Transaction) compare.Order {
	if o := compare.Time(t.Date, t2.Date); o != compare.Equal {
		return o
	}
	if o := compare.Ordered(t.Description, t2.Description); o != compare.Equal {
		return o
	}
	for i := 0; i < len(t.Postings) && i < len(t2.Postings); i++ {
		if o := posting.Compare(t.Postings[i], t2.Postings[i]); o != compare.Equal {
			return o
		}
	}
	return compare.Ordered(len(t.Postings), len(t2.Postings))
}

// Builder builds transactions.
type Builder struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*posting.Posting
	Targets     []*commodity.Commodity
}

// Build builds a transactions.
func (tb Builder) Build() *Transaction {
	return &Transaction{
		Src:         tb.Src,
		Date:        tb.Date,
		Description: tb.Description,
		Postings:    tb.Postings,
		Targets:     tb.Targets,
	}
}

func Create(reg *registry.Registry, t *syntax.Transaction) *Transaction {
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
