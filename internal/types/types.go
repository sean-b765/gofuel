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

type GoogleJourneyResponse struct {
	Distance string
	Duration string
}
