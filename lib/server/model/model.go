package model

import "time"

// Commodity represents a commodity.
type Commodity struct {
	ID   int
	Name string
}

// Version represents a data version.
type Version struct {
	Version     int
	Description string
	CreatedAt   time.Time
}

// Account represents an account.
type Account struct {
	ID        int
	Name      string
	OpenDate  time.Time
	CloseDate *time.Time
}
