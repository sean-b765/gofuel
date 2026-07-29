package types

type Station struct {
	Id          string
	Title       string
	Brand       string
	Date        string
	Price       FuelPrice
	TradingName string
	Location    string
	Address     string
	Latitude    float64
	Longitude   float64
	DistanceTo  float64
}

type FuelPrice struct {
	Ulp91  float32
	Ulp95  float32
	Ulp98  float32
	Diesel float32
}

type JsonResponse struct {
	Stations []Station
}

type StationItem struct {
	PK        string // geohash precision 1
	SK        string // geohash precision 8
	GSI1PK    string // geohash precision 4
	GSI1SK    string // geohash precision 8
	StationId string
	Title     string
	Brand     string
	Address   string
	Latitude  float64
	Longitude float64
	Ulp91     float32
	Ulp95     float32
	Ulp98     float32
	Diesel    float32
	Date      string
}

type GoogleJourneyResponse struct {
	Distance string
	Duration string
}
