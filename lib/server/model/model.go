package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// CommodityID is the ID of a commodity.
type CommodityID int

// Commodity represents a commodity.
type Commodity struct {
	ID   CommodityID
	Name string
}

// Less defines an ordering on the price directive. It has no semantics,
// but can be used in tests, for example.
func (p Commodity) Less(p1 Commodity) bool {
	return p.ID < p1.ID
}

// Price represents a price.
type Price struct {
	Date                           time.Time
	CommodityID, TargetCommodityID CommodityID
	Price                          float64
}

// Less defines an ordering on the price directive. It has no semantics,
// but can be used in tests, for example.
func (p Price) Less(p1 Price) bool {
	if p.CommodityID != p1.CommodityID {
		return p.CommodityID < p1.CommodityID
	}
	if p.TargetCommodityID != p1.TargetCommodityID {
		return p.TargetCommodityID < p1.TargetCommodityID
	}
	return p.Date.Before(p1.Date)
}

// AccountID is the ID of an account.
type AccountID int

// Account represents an account.
type Account struct {
	ID        AccountID
	Name      string
	OpenDate  time.Time
	CloseDate *time.Time
}

// Less defines an ordering on Account.
func (a Account) Less(a2 Account) bool {
	return a.ID < a2.ID
}

// Transaction represents a transaction.
type Transaction struct {
	ID          TransactionID
	Date        time.Time
	Description string
	Bookings    []Booking
}

// Less defines an ordering on Transaction.
func (t Transaction) Less(t2 Transaction) bool {
	return t.ID < t2.ID
}

// TransactionID is the ID of a booking.
type TransactionID int

// Booking represents a booking.
type Booking struct {
	Amount                          decimal.Decimal
	CommodityID                     CommodityID
	CreditAccountID, DebitAccountID AccountID
}

// Less defines an ordering on Transaction.
func (b Booking) Less(b2 Booking) bool {
	if b.CommodityID != b2.CommodityID {
		return b.CommodityID < b2.CommodityID
	}
	if b.CreditAccountID != b2.CreditAccountID {
		return b.CreditAccountID < b2.CreditAccountID
	}
	if b.DebitAccountID != b2.DebitAccountID {
		return b.DebitAccountID < b2.DebitAccountID
	}
	return b.Amount.LessThan(b2.Amount)
}
