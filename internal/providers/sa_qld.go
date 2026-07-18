// Fuel Pricing Information Scheme (SA, QLD)

package providers

type FpisStationPrice struct {
	SiteId             int
	FuelId             int
	CollectionMethod   string
	TransactionDateUtc string
	Price              float32
}

type FpisStation struct {
	// SiteId
	S int
	// Address
	A string
	// Name
	N   string
	Lat float64
	Lng float64
}
