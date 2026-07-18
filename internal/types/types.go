package types

type Rss struct {
	Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Items       []Item `xml:"item"`
	} `xml:"channel"`
}

type Item struct {
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

type Journey struct {
	Rows []struct {
		Elements []struct {
			Distance struct {
				Text  string `json:"text"`
				Value int    `json:"value"`
			} `json:"distance"`
			Duration struct {
				Text  string `json:"text"`
				Value int    `json:"value"`
			} `json:"duration"`
			Status string `json:"status"`
		} `json:"elements"`
	} `json:"rows"`
	Status string `json:"status"`
}

type JsonResponse struct {
	Date     string
	Stations []Item
}

type JourneyJsonResponse struct {
	Distance string
	Duration string
}