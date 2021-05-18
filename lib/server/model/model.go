package model

import "time"

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
