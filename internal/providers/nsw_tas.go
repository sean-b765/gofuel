// NSW/TAS FuelAPI product

package providers

import (
	"strconv"

	"seanboaden.dev/fuel/internal/types"
)

type NswTasStation struct {
	brandid   string
	stationid string
	brand     string
	code      int
	name      string
	address   string
	state     string
	location  location
}

type location struct {
	latitude  int
	longitude int
}

type NswTasStationFuelPrice struct {
	stationcode int
	fueltype    string
	price       float32
	priceunit   string
	lastupdated string
	state       string
}

func TransformNswTasStations(stations []NswTasStation, prices []NswTasStationFuelPrice) []types.Station {
	pricesByStation := map[int][]NswTasStationFuelPrice{}
	for _, p := range prices {
		pricesByStation[p.stationcode] = append(pricesByStation[p.stationcode], p)
	}

	result := make([]types.Station, 0, len(stations))
	for _, s := range stations {
		fuelPrices := types.FuelPrice{}
		date := ""
		for _, p := range pricesByStation[s.code] {
			switch p.fueltype {
			case "U91":
				fuelPrices.Ulp91 = p.price
				break
			case "P95":
				fuelPrices.Ulp95 = p.price
				break
			case "P98":
				fuelPrices.Ulp98 = p.price
				break
			default:
				continue
			}
			date = p.lastupdated
		}

		result = append(result, types.Station{
			Id:        strconv.Itoa(s.code),
			Title:     s.name,
			Brand:     s.brand,
			Date:      date,
			Location:  s.state,
			Address:   s.address,
			Latitude:  float64(s.location.latitude),
			Longitude: float64(s.location.longitude),
			Price:     fuelPrices,
		})
	}
	return result
}

func GetNswTasPricesCurrent() []types.Station {
	return TransformNswTasStations(nil, nil)
}
