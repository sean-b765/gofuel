// NSW/TAS FuelAPI product

package providers

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
