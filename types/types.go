package types

type Rss struct {
	Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Items       []Item `xml:"item"`
	} `xml:"channel"`
}

type Item struct {
	Title           string `xml:"title"`
	Brand           string `xml:"brand"`
	Date            string `xml:"date"`
	Price           string `xml:"price"`
	TradingName     string `xml:"trading-name"`
	Location        string `xml:"location"`
	Address         string `xml:"address"`
	Phone           string `xml:"phone"`
	Latitude        string `xml:"latitude"`
	Longitude       string `xml:"longitude"`
	DistanceTo      float64
	JourneyTime     string
	JourneyDistance string
}
