// Fuel Pricing Information Scheme (SA, QLD)

package providers

import (
	"strconv"

	"seanboaden.dev/fuel/internal/types"
)

type SaQldStationPrice struct {
	SiteId             int
	FuelId             int
	CollectionMethod   string
	TransactionDateUtc string
	Price              float32
}

type SaQldStation struct {
	// SiteId
	S int
	// Address
	A string
	// Name
	N   string
	Lat float64
	Lng float64
}

func TransformSaQldStations(stations []SaQldStation, prices []SaQldStationPrice) []types.Station {
	pricesBySite := map[int][]SaQldStationPrice{}
	for _, p := range prices {
		pricesBySite[p.SiteId] = append(pricesBySite[p.SiteId], p)
	}

	result := make([]types.Station, 0, len(stations))
	for _, s := range stations {
		fuelPrice := types.FuelPrice{}
		date := ""
		for _, p := range pricesBySite[s.S] {
			switch p.FuelId {
			case 2:
				fuelPrice.Ulp91 = p.Price
				break
			case 5:
				fuelPrice.Ulp95 = p.Price
				break
			case 8:
				fuelPrice.Ulp98 = p.Price
				break
			default:
				continue
			}
			date = p.TransactionDateUtc
		}

		result = append(result, types.Station{
			Id:        strconv.Itoa(s.S),
			Title:     s.N,
			Date:      date,
			Address:   s.A,
			Latitude:  s.Lat,
			Longitude: s.Lng,
			Price:     fuelPrice,
		})
	}
	return result
}

func GetSaQldPricesCurrent() []types.Station {
	return TransformSaQldStations(nil, nil)
}
