package providers

import (
	"encoding/xml"
	"io"
	"net/http"

	"seanboaden.dev/fuel/internal/types"
)

type rss struct {
	Channel struct {
		Title       string          `xml:"title"`
		Description string          `xml:"description"`
		Items       []types.Station `xml:"item"`
	} `xml:"channel"`
}

// FuelWatch API from WA
// Get the latest fuel prices. Returns the item array and the date
func GetWaPricesCurrent() ([]types.Station, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}

func GetWaPricesTomorrow() ([]types.Station, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?day=tomorrow")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}
