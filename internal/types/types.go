package types

type Station struct {
	Id        string
	Title     string
	Brand     string
	Date      string
	Price     FuelPrice
	Address   string
	Latitude  float64
	Longitude float64
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
	RegionGeohash    string  `json:"region_geohash"`     // PK (precision 1)
	TownGeohash      string  `json:"town_geohash"`       // SK / GSI1SK (precision 8)
	SubRegionGeohash string  `json:"sub_region_geohash"` // GSI1PK (precision 4)
	StationId        string  `json:"station_id"`
	Title            string  `json:"title"`
	Brand            string  `json:"brand"`
	Address          string  `json:"address"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Ulp91            float32 `json:"ulp91"`
	Ulp95            float32 `json:"ulp95"`
	Ulp98            float32 `json:"ulp98"`
	Diesel           float32 `json:"diesel"`
	Date             string  `json:"date"`
	TTL              int64   `json:"-" dynamodbav:"ttl"`
}

type GoogleJourneyResponse struct {
	Distance string
	Duration string
}
