package macro

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/model/accounts"
)

// Expand expands an accrual transaction.
func Expand(accrual *ledger.Accrual) ([]*ledger.Transaction, error) {
	t := accrual.Transaction
	if l := len(t.Postings); l != 1 {
		return nil, fmt.Errorf("%s: accrual expansion: number of postings is %d, must be 1", accrual.Transaction.Position().Start, l)
	}
	posting := t.Postings[0]
	var (
		crAccountSingle, drAccountSingle, crAccountMulti, drAccountMulti = accrual.Account, accrual.Account, accrual.Account, accrual.Account
	)
	switch {
	case isAL(posting.Credit) && isIE(posting.Debit):
		crAccountSingle = posting.Credit
		drAccountMulti = posting.Debit
	case isIE(posting.Credit) && isAL(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountSingle = posting.Debit
	case isIE(posting.Credit) && isIE(posting.Debit):
		crAccountMulti = posting.Credit
		drAccountMulti = posting.Debit
	default:
		crAccountSingle = posting.Credit
		drAccountSingle = posting.Debit
	}
	amount := posting.Amount.Amount()
	dates := date.Series(accrual.T0, accrual.T1, accrual.Period)[1:]
	rated, rem := amount.QuoRem(decimal.NewFromInt(int64(len(dates))), 1)

	var result []*ledger.Transaction
	if crAccountMulti != drAccountMulti {
		for i, date := range dates {
			a := rated
			if i == 0 {
				a = a.Add(rem)
			}
			result = append(result, &ledger.Transaction{
				Pos:         t.Pos,
				Date:        date,
				Tags:        t.Tags,
				Description: fmt.Sprintf("%s (accrual %d/%d)", t.Description, i+1, len(dates)),
				Postings: []*ledger.Posting{
					ledger.NewPosting(crAccountMulti, drAccountMulti, posting.Commodity, a),
				},
			})
		}
	}
	if crAccountSingle != drAccountSingle {
		result = append(result, &ledger.Transaction{
			Pos:         t.Pos,
			Date:        t.Date,
			Tags:        t.Tags,
			Description: t.Description,
			Postings: []*ledger.Posting{
				ledger.NewPosting(crAccountSingle, drAccountSingle, posting.Commodity, posting.Amount.Amount()),
			},
		})

	}
	return result, nil
}

func isAL(a *accounts.Account) bool {
	return a.Type() == accounts.ASSETS || a.Type() == accounts.LIABILITIES
}

func isIE(a *accounts.Account) bool {
	return a.Type() == accounts.INCOME || a.Type() == accounts.EXPENSES
}
