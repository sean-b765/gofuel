package providers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
)

type WaStation struct {
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
}

type rss struct {
	Channel struct {
		Title       string      `xml:"title"`
		Description string      `xml:"description"`
		Items       []WaStation `xml:"item"`
	} `xml:"channel"`
}

// Create an id for deduping station data
func Md5HashWa(station WaStation) string {
	dataString := fmt.Sprintf("%s|%f|%f", station.Address, util.ToFloat(station.Latitude), util.ToFloat(station.Longitude))
	hashArray := md5.Sum([]byte(dataString))
	return hex.EncodeToString(hashArray[:])
}

func transformWaStations(items []WaStation) []types.Station {
	stations := make([]types.Station, 0, len(items))
	for _, item := range items {
		stations = append(stations, types.Station{
			Id:          Md5HashWa(item),
			Title:       item.Title,
			Brand:       item.Brand,
			Date:        item.Date,
			TradingName: item.TradingName,
			Location:    item.Location,
			Address:     item.Address,
			Latitude:    util.ToFloat(item.Latitude),
			Longitude:   util.ToFloat(item.Longitude),
			Price:       types.FuelPrice{Ulp91: float32(util.ToFloat(item.Price))},
		})
	}
	return stations
}

// FuelWatch API from WA
// Get the latest fuel prices. Returns the item array and the date
func GetWaPricesCurrent() []types.Station {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response rss
	xml.Unmarshal(byteValue, &response)

	return transformWaStations(response.Channel.Items)
}

func GetWaPricesTomorrow() ([]WaStation, string) {
	resp, err := http.Get("https://www.fuelwatch.wa.gov.au/fuelwatch/fuelWatchRSS?day=tomorrow")
	if err != nil {
		panic("Error with http.Get")
	}

	byteValue, _ := io.ReadAll(resp.Body)

	var response rss
	xml.Unmarshal(byteValue, &response)

	return response.Channel.Items, response.Channel.Description
}
