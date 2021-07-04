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

// Price represents a price.
type Price struct {
	Date                           time.Time
	CommodityID, TargetCommodityID CommodityID
	Price                          float64
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

// Transaction represents a transaction.
type Transaction struct {
	ID          TransactionID
	Date        time.Time
	Description string
	Bookings    []Booking
}

// TransactionID is the ID of a booking.
type TransactionID int

// Booking represents a booking.
type Booking struct {
	ID                              TransactionID
	Amount                          decimal.Decimal
	CommodityID                     CommodityID
	CreditAccountID, DebitAccountID AccountID
}
