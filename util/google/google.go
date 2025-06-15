package google

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/tidwall/gjson"
)

func GetJourney(origin, destination string) (string, string) {
	url := "https://maps.googleapis.com/maps/api/distancematrix/json?origins=" + origin + "&destinations=" + destination + "&mode=driving&language=en-US&key=" + os.Getenv("MAPS_KEY")
	fmt.Println(url)
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	byteValue, _ := io.ReadAll(resp.Body)

	response := gjson.ParseBytes(byteValue)

	// Error message provided
	status := response.Get("status").String()
	if status != "OK" {
		error_message := response.Get("error_message").String()
		fmt.Println(status + " : " + error_message)
		return "", ""
	}

	rows := response.Get("rows")

	if !rows.Exists() {
		panic(errors.New("no rows received from google maps api"))
	}

	distance := rows.Array()
	duration := rows.Array()
	if len(distance) == 0 || len(duration) == 0 {
		return "", ""
	}

	distanceArray := distance[0].Get("elements.#.distance.text").Array()
	durationArray := duration[0].Get("elements.#.duration.text").Array()

	if len(distanceArray) == 0 || len(durationArray) == 0 {
		return "", ""
	}

	return distanceArray[0].String(), durationArray[0].String()
}