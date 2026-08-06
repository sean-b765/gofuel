// Fuel Pricing Information Scheme (SA, QLD)

package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"seanboaden.dev/fuel/internal/secrets"
	"seanboaden.dev/fuel/internal/types"
	"seanboaden.dev/fuel/internal/util"
)

type SaQldStationPrice struct {
	SiteId             int     `json:"SiteId"`
	FuelId             int     `json:"FuelId"`
	CollectionMethod   string  `json:"CollectionMethod"`
	TransactionDateUtc string  `json:"TransactionDateUtc"`
	Price              float32 `json:"Price"`
}

type SaQldStation struct {
	S   int     `json:"S"`
	A   string  `json:"A"`
	N   string  `json:"N"`
	Lat float64 `json:"Lat"`
	Lng float64 `json:"Lng"`
}

type saQldSitesResponse struct {
	S []SaQldStation `json:"S"`
}

type saQldPricesResponse struct {
	SitePrices []SaQldStationPrice `json:"SitePrices"`
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
				fuelPrice.Ulp91 = p.Price / 10
			case 3:
				fuelPrice.Diesel = p.Price / 10
			case 5:
				fuelPrice.Ulp95 = p.Price / 10
			case 8:
				fuelPrice.Ulp98 = p.Price / 10
			default:
				continue
			}
			date = util.FormatDate("2006-01-02T15:04:05", p.TransactionDateUtc)
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

func fetchSaQld(baseURL string, apiKey string, geoRegionLevel, geoRegionId int) []types.Station {
	query := "?countryId=21&geoRegionLevel=" + strconv.Itoa(geoRegionLevel) +
		"&geoRegionId=" + strconv.Itoa(geoRegionId)

	var sites saQldSitesResponse
	if !getSaQldJson(baseURL+"/Subscriber/GetFullSiteDetails"+query, apiKey, &sites) {
		return TransformSaQldStations(nil, nil)
	}

	var prices saQldPricesResponse
	if !getSaQldJson(baseURL+"/Price/GetSitesPrices"+query, apiKey, &prices) {
		return TransformSaQldStations(nil, nil)
	}

	return TransformSaQldStations(sites.S, prices.SitePrices)
}

func getSaQldJson(url string, apiKey string, out any) bool {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	if err := json.Unmarshal(byteValue, out); err != nil {
		return false
	}

	return true
}

func GetSaQldPricesCurrent() []types.Station {
	result := []types.Station{}

	result = append(result, fetchSaQld(
		"https://fppdirectapi-prod.safuelpricinginformation.com.au",
		secrets.Get("SA_API_KEY"),
		2,
		189,
	)...)

	result = append(result, fetchSaQld(
		"https://fppdirectapi-prod.fuelpricesqld.com.au",
		secrets.Get("QLD_API_KEY"),
		3,
		1,
	)...)

	return result
}
