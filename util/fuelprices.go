/*
	Contains utility functions
*/

package util

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"

	"example.com/fuel/types"
)

// Get the latest fuel prices. Returns the item array and the date
func GetFuelPrices() ([]types.Item, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := ioutil.ReadAll(resp.Body)

	var response types.Rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}
