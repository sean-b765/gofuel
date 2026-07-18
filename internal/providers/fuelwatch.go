package providers

import (
	"encoding/xml"
	"io"
	"net/http"

	"seanboaden.dev/fuel/internal/types"
)

// FuelWatch API from WA
// Get the latest fuel prices. Returns the item array and the date
func GetWaPricesCurrent() ([]types.Item, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response types.Rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}

func GetWaPricesTomorrow() ([]types.Item, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?day=tomorrow")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response types.Rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}
