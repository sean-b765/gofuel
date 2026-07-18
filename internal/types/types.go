package types

type Station struct {
	Title       string `xml:"title"`
	Brand       string `xml:"brand"`
	Date        string `xml:"date"`
	Price       string `xml:"price"`
	TradingName string `xml:"trading-name"`
	Location    string `xml:"location"`
	Address     string `xml:"address"`
	Phone       string `xml:"phone"`
	Latitude    string `xml:"latitude"`
	Longitude   string `xml:"longitude"`
	DistanceTo  float64
}

type JsonResponse struct {
	Date     string
	Stations []Station
}

type GoogleJourneyResponse struct {
	Distance string
	Duration string
}
