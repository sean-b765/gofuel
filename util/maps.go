package util

import (
	"io"
	"net/http"
	"os"

	"github.com/tidwall/gjson"
)

func GetJourney(origin, destination string) (string, string) {
	url := "https://maps.googleapis.com/maps/api/distancematrix/json?origins=" + origin + "&destinations=" + destination + "&mode=driving&language=en-US&key=" + os.Getenv("MAPS_KEY")
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	byteValue, _ := io.ReadAll(resp.Body)

	response := gjson.ParseBytes(byteValue)

	distance := response.Get("rows").Array()[0].Get("elements.#.distance.text").Array()[0].String()
	duration := response.Get("rows").Array()[0].Get("elements.#.duration.text").Array()[0].String()

	return distance, duration
}
