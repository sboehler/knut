package model

import (
	"time"

	"github.com/sboehler/knut/lib/syntax"
	"github.com/shopspring/decimal"
)

type PostingBuilder struct {
	Src           *syntax.Booking
	Amount, Value decimal.Decimal
	Credit, Debit *Account
	Commodity     *Commodity
}

func (pb PostingBuilder) Build() []*Posting {
	if pb.Amount.IsNegative() || pb.Amount.IsZero() && pb.Value.IsNegative() {
		pb.Credit, pb.Debit, pb.Amount, pb.Value = pb.Debit, pb.Credit, pb.Amount.Neg(), pb.Value.Neg()
	}
	return []*Posting{
		{
			Src:       pb.Src,
			Account:   pb.Credit,
			Other:     pb.Debit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount.Neg(),
			Value:     pb.Value.Neg(),
		},
		{
			Src:       pb.Src,
			Account:   pb.Debit,
			Other:     pb.Credit,
			Commodity: pb.Commodity,
			Amount:    pb.Amount,
			Value:     pb.Value,
		},
	}
}

type PostingBuilders []PostingBuilder

func (pbs PostingBuilders) Build() []*Posting {
	res := make([]*Posting, 0, 2*len(pbs))
	for _, pb := range pbs {
		res = append(res, pb.Build()...)
	}
	return res
}

// TransactionBuilder builds transactions.
type TransactionBuilder struct {
	Src         *syntax.Transaction
	Date        time.Time
	Description string
	Postings    []*Posting
	Targets     []*Commodity
}

// Build builds a transactions.
func (tb TransactionBuilder) Build() *Transaction {
	return &Transaction{
		Src:         tb.Src,
		Date:        tb.Date,
		Description: tb.Description,
		Postings:    tb.Postings,
		Targets:     tb.Targets,
	}
}
