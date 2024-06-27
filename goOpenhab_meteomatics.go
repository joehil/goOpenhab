package main

import (
	"context"
	"log"
	"time"

	"github.com/twpayne/go-meteomatics"
)

func getWeather() string {
	var weather [17]string
	weather[0] = "No weather"
	weather[1] = "Clear sky"
	weather[2] = "Light clouds"
	weather[3] = "Partly cloudy"
	weather[4] = "Cloudy"
	weather[5] = "Rain"
	weather[6] = "Rain and snow / sleet"
	weather[7] = "Snow"
	weather[8] = "Rain shower"
	weather[9] = "Snow shower"
	weather[10] = "Sleet shower"
	weather[11] = "Light Fog"
	weather[12] = "Dense fog"
	weather[13] = "Freezing rain"
	weather[14] = "Thunderstorms"
	weather[15] = "Drizzle"
	weather[16] = "Sandstorm"

	var retWeather string = ""

	client := meteomatics.NewClient(
		meteomatics.WithBasicAuth(
			genVar.MMuserid,
			genVar.MMpassw,
		),
	)

	cr, err := client.RequestCSV(
		context.Background(),
		meteomatics.TimeRange{
			Start: time.Now(),
			End:   time.Now().AddDate(0, 0, 2),
			Step:  1 * time.Hour,
		},
		meteomatics.Parameter{
			Name:  "weather_symbol_1h",
			Units: "idx",
		},
		meteomatics.Postal{
			CountryCode: genVar.MMcountry,
			ZIPCode:     genVar.MMpostcode,
		},
		&meteomatics.RequestOptions{},
	)
	if err != nil {
		log.Println(err)
		return "<error>"
	}

	log.Println(cr.Parameters)
	for _, row := range cr.Rows {
		//	debugLog(6,row.ValidDate)
		x := int(row.Values[0])
		if x > 99 {
			x -= 100
		}
		debugLog(6, weather[x])
		retWeather += ">" + weather[x]
	}
	return retWeather
}
