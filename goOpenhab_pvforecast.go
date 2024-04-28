package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Define structs that correspond to the JSON structure
type Data struct {
	Result  Result  `json:"result"`
	Message Message `json:"message"`
}

type Result struct {
	Watts           map[string]int `json:"watts"`
	WattHoursPeriod map[string]int `json:"watt_hours_period"`
	WattHours       map[string]int `json:"watt_hours"`
	WattHoursDay    map[string]int `json:"watt_hours_day"`
}

type Message struct {
	Code      int       `json:"code"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	PID       string    `json:"pid"`
	Info      Info      `json:"info"`
	RateLimit RateLimit `json:"ratelimit"`
}

type Info struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Distance  int       `json:"distance"`
	Place     string    `json:"place"`
	Timezone  string    `json:"timezone"`
	Time      time.Time `json:"time"`
	TimeUTC   time.Time `json:"time_utc"`
}

type RateLimit struct {
	Zone      string `json:"zone"`
	Period    int    `json:"period"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
}

func getPvForecast() {
	var requrl string
	if genVar.PVApiToken == "" {
		requrl = genVar.PVurl + "/estimate/" + genVar.PVlatitude + "/" + genVar.PVlongitude + "/" +
			genVar.PVdeclination + "/" + genVar.PVazimuth + "/" + genVar.PVkw
	} else {
		requrl = genVar.PVurl + "/" + genVar.PVApiToken + "/estimate/" + genVar.PVlatitude + "/" + genVar.PVlongitude + "/" +
			genVar.PVdeclination + "/" + genVar.PVazimuth + "/" + genVar.PVkw
	}

	for {
		// Erstelle einen neuen Request
		req, err := http.NewRequest("GET", requrl, nil)
		if err != nil {
			traceLog(fmt.Sprintf("pvforecast get creation error: %v", err))
			createMessage("pvforecast.creation.event", fmt.Sprintf("%v", err), "")
		}

		// Erstelle einen neuen HTTP-Client und führe den Request aus
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("pvforecast get processing error: %v", err))
			createMessage("pvforecast.processing.error.event", fmt.Sprintf("%v", err), "")

		} else {

			// Lies den Response Body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				traceLog(fmt.Sprintf("pvforecast get error reading response: %v", err))
				createMessage("pvforecast.get.error.event", fmt.Sprintf("%v", err), "")

			} else {
				// Gib den Response Body aus
				debugLog(5, fmt.Sprintf("pvforecast get received response: %v", string(body)))

				// Unmarshal the JSON data
				var data Data
				err := json.Unmarshal(body, &data)
				if err != nil {
					traceLog(fmt.Sprintf("pvforecast JSON unmarshaling error: %v", err))
					createMessage("pvforecast.json.error.event", fmt.Sprintf("%v", err), "")
				}

				for date, wattHours := range data.Result.WattHoursDay {
					debugLog(5, fmt.Sprintf("pvforecast watthours: %v %d", date, wattHours))
					createMessage("pvforecast.watthours.event", date, fmt.Sprintf("%d", wattHours))
				}
			}

			resp.Body.Close()
		}
		time.Sleep(30 * time.Minute)
	}
}
