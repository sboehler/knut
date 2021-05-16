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
