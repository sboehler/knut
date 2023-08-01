package commodity

// Commodity represents a currency or security.
type Commodity struct {
	name       string
	IsCurrency bool
}

func (c Commodity) Name() string {
	return c.name
}

func (c Commodity) String() string {
	return c.name
}
