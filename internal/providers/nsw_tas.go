// NSW/TAS FuelAPI product

package providers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	"seanboaden.dev/fuel/internal/auth"
	"seanboaden.dev/fuel/internal/types"
)

type NswTasStation struct {
	Brandid   string   `json:"brandid"`
	Stationid string   `json:"stationid"`
	Brand     string   `json:"brand"`
	Code      int      `json:"code"`
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	State     string   `json:"state"`
	Location  Location `json:"location"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type NswTasStationFuelPrice struct {
	Stationcode int     `json:"stationcode"`
	Fueltype    string  `json:"fueltype"`
	Price       float64 `json:"price"`
	Priceunit   string  `json:"priceunit"`
	Lastupdated string  `json:"lastupdated"`
	State       string  `json:"state"`
}

type nswTasResponse struct {
	Stations []NswTasStation          `json:"stations"`
	Prices   []NswTasStationFuelPrice `json:"prices"`
}

func TransformNswTasStations(stations []NswTasStation, prices []NswTasStationFuelPrice) []types.Station {
	pricesByStation := map[int][]NswTasStationFuelPrice{}
	for _, p := range prices {
		pricesByStation[p.Stationcode] = append(pricesByStation[p.Stationcode], p)
	}

	result := make([]types.Station, 0, len(stations))
	for _, s := range stations {
		fuelPrices := types.FuelPrice{}
		date := ""
		for _, p := range pricesByStation[s.Code] {
			switch p.Fueltype {
			case "U91":
				fuelPrices.Ulp91 = float32(p.Price)
				break
			case "P95":
				fuelPrices.Ulp95 = float32(p.Price)
				break
			case "P98":
				fuelPrices.Ulp98 = float32(p.Price)
				break
			default:
				continue
			}
			date = p.Lastupdated
		}

		result = append(result, types.Station{
			Id:        strconv.Itoa(s.Code),
			Title:     s.Name,
			Brand:     s.Brand,
			Date:      date,
			Location:  s.State,
			Address:   s.Address,
			Latitude:  s.Location.Latitude,
			Longitude: s.Location.Longitude,
			Price:     fuelPrices,
		})
	}
	return result
}

func GetNswTasPricesCurrent() []types.Station {
	token, err := auth.GetNswTasToken()
	if err != nil {
		return TransformNswTasStations(nil, nil)
	}

	req, err := http.NewRequest("GET", "https://api.onegov.nsw.gov.au/FuelPriceCheck/v2/fuel/prices", nil)
	if err != nil {
		return TransformNswTasStations(nil, nil)
	}

	req.Header.Set("apikey", os.Getenv("NSW_TAS_API_KEY"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("transactionid", uuid.NewString())
	req.Header.Set("requesttimestamp", time.Now().Format("02/01/2006 3:04:05 PM"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TransformNswTasStations(nil, nil)
	}
	defer resp.Body.Close()

	var body nswTasResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return TransformNswTasStations(nil, nil)
	}

	return TransformNswTasStations(body.Stations, body.Prices)
}
