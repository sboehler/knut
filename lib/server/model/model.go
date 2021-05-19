package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Commodity represents a commodity.
type Commodity struct {
	ID   int
	Name string
}

// Price represents a price.
type Price struct {
	Date                           time.Time
	CommodityID, TargetCommodityID int
	Price                          float64
}

// Account represents an account.
type Account struct {
	ID        int
	Name      string
	OpenDate  time.Time
	CloseDate *time.Time
}

// Transaction represents a transaction.
type Transaction struct {
	ID          int
	Date        time.Time
	Description string
	Bookings    []Booking
}

// Booking represents a booking.
type Booking struct {
	ID                              int
	Amount                          decimal.Decimal
	CommodityID                     int
	CreditAccountID, DebitAccountID int
}
