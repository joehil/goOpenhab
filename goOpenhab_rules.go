package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

func processRulesInfo(mInfo Msginfo) {
	if mInfo.Msgevent == "chrono.event" {
		chronoEvents(mInfo)
		return
	}

	// store every Tibber variable in cache
	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Tibber_t" || mInfo.Msgobject[0:8] == "Tibber_m" || mInfo.Msgobject[0:8] == "Tibber_n" {
			fmt.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		}
	}

	// store the SOC of our battery in cache
	if mInfo.Msgobject == "Solarakku_SOC" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		fmt.Println("SOC stored: ", mInfo.Msgnewstate)
		return
	}

	// inform about sunrise
	if (mInfo.Msgobject == "astro:sun:local:rise#event") &&
		(mInfo.Msgnewstate == "END") {
		genVar.Telegram <- "Sonnenaufgang"
		return
	}

	// inform about sunset
	if (mInfo.Msgobject == "astro:sun:local:set#event") &&
		(mInfo.Msgnewstate == "END") {
		genVar.Telegram <- "Sonnenuntergang"
		return
	}
}

func chronoEvents(mInfo Msginfo) {
	// this rule runs every minute
	var batPrice string
	ap := getItemState("curr_price")
	soc := getItemState("Solarakku_SOC")
	stromGarage := getItemState("Balkonkraftwerk_Garage_Stromproduktion")
	stromBalkon := getItemState("Schalter_Balkon_Power")
	as := getItemState("Tibber_Aktueller_Verbrauch")
	fmt.Printf("Current price: %s, SOC: %s, Strom Garage: %s, Strom Balkon %s, Aktueller Verbrauch %s\n", ap, soc, stromGarage, stromBalkon, as)
	flStromGarage, _ := strconv.ParseFloat(stromGarage, 64)
	flStromBalkon, _ := strconv.ParseFloat(stromBalkon, 64)
	//	flAs, _ := strconv.ParseFloat(as, 64)
	var flStrom float64 = flStromGarage + flStromBalkon
	x, found := genVar.Pers.Get("!BAT_PRICE")
	if found && flStrom < float64(50) {
		batPrice = x.(string)
		fmt.Println("BAT_PRICE: ", batPrice)
		if soc > "35.00" && ap >= batPrice {
			// Soyosource on
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set/state", Message: "on"}
			fmt.Println("Soyosource switched on")
		} else {
			// Soyosource off
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set/state", Message: "off"}
			fmt.Println("Soyosource switched off")
		}
	} else {
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set/state", Message: "off"}
		fmt.Println("Soyosource switched off")
	}
	if flStrom > float64(100) {
		// Loader-48 on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set/state", Message: "on"}
		fmt.Println("Laden_48 switched on")
	} else {
		// Loader_48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set/state", Message: "off"}
		fmt.Println("Laden_48 switched off")
	}

	// this rule runs at minutes 3, 6 and 9
	if strings.ContainsAny(mInfo.Msgobject[4:5], "369") {
		item := "Tibber_total" + mInfo.Msgobject[0:2]
		fmt.Println(item)
		if x, found := genVar.Pers.Get(item); found {
			foo := x.(string)
			fmt.Println(foo)
		}
		return
	}

	// this rules runs at the first minute of each hour
	if mInfo.Msgobject[3:5] == "00" {
		item := "Tibber_total" + mInfo.Msgobject[0:2]
		if mInfo.Msgobject[0:2] == "00" {
			item = "Tibber_tomorrow00"
		}
		fmt.Println(item)
		genVar.Getin <- Requestin{Node: "items", Item: item, Value: "state"}
		answer := <-genVar.Getout
		fmt.Println(answer)
		if answer != "" {
			genVar.Putin <- Requestin{Node: "items", Item: "curr_price", Value: "state", Data: answer}
		}
		calculateBatteryPrice(mInfo.Msgobject[0:2])
		return
	}
}

// rules that are called when goOpenhab initializes

func rulesInit() {
	now := time.Now()
	hour := now.Hour()
	calculateBatteryPrice(fmt.Sprintf("%02d", hour))
}

// special funtions as a support to make relatively short rules

func calculateBatteryPrice(hour string) {
	var flSoc float64
	var prices []float64
	var price string
	var flPrice float64
	soc := getItemState("Solarakku_SOC")
	flSoc, _ = strconv.ParseFloat(soc, 64)
	flSoc -= 30
	hours := int(float64(flSoc / 8))
	intH, _ := strconv.Atoi(hour)
	for i := intH; i < 24; i++ {
		price = getItemState(fmt.Sprintf("Tibber_total%02d", i))
		flPrice, _ = strconv.ParseFloat(price, 64)
		if flPrice > float64(0.21) {
			prices = append(prices, flPrice)
		}
	}
	if hour > "15" {
		for i := 0; i < 10; i++ {
			price = getItemState(fmt.Sprintf("Tibber_tomorrow%02d", i))
			flPrice, _ = strconv.ParseFloat(price, 64)
			if flPrice > float64(0.21) {
				prices = append(prices, flPrice)
			}
		}
	}
	sort.Float64s(prices)
	lPrices := len(prices)
	lPrices -= hours
	if lPrices >= 0 {
		price = fmt.Sprintf("%0.4f", prices[lPrices])
	} else {
		price = "9.9999"
	}
	fmt.Println("Bat-Price: ", price, hours)
	fmt.Println(prices)
	genVar.Pers.Set("!BAT_PRICE", price, cache.DefaultExpiration)
}
