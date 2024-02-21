package main

import (
	"fmt"
	"strings"
	"strconv"
	"sort"
	"github.com/patrickmn/go-cache"
)

func processRulesInfo(mInfo Msginfo) {
	if mInfo.Msgevent == "chrono.event" {
		chronoEvents(mInfo)
		return
	}

	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Tibber_t" || mInfo.Msgobject[0:8] == "Tibber_m" || mInfo.Msgobject[0:8] == "Tibber_n" {
			fmt.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		}
	}
	if mInfo.Msgobject == "Solarakku_SOC" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		fmt.Println("SOC stored: ", mInfo.Msgnewstate)
		return
	}
	if (mInfo.Msgobject == "astro:sun:local:rise#event") &&
		(mInfo.Msgnewstate == "END") {
		genVar.Telegram <- "Sonnenaufgang"
		return
	}
	if (mInfo.Msgobject == "astro:sun:local:set#event") &&
		(mInfo.Msgnewstate == "END") {
		genVar.Telegram <- "Sonnenuntergang"
		return
	}
}

func chronoEvents(mInfo Msginfo) {
	var batPrice string
	m2 := getItemState("Tibber_m2")
	ap := getItemState("curr_price")
	fmt.Println("m2: ",m2)
	fmt.Println("ap: ",ap)
	soc := getItemState("Solarakku_SOC")
	fmt.Println("SOC found: ", soc)
	if x, found := genVar.Pers.Get("!BAT_PRICE"); found {
                batPrice= x.(string)
                fmt.Println("BAT_PRICE: ",batPrice)
		if soc > "35.00" && ap > batPrice {
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set/state", Message: "on"}
			fmt.Println("Soyosource switched on")
		} else {
                        genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set/state", Message: "off"}
			fmt.Println("Soyosource switched off")
		}
	}

	if strings.ContainsAny(mInfo.Msgobject[4:5], "369") {
		item := "Tibber_total" + mInfo.Msgobject[0:2]
		fmt.Println(item)
		//		genVar.Pers.Set(item, "jhtest", cache.DefaultExpiration)
		if x, found := genVar.Pers.Get(item); found {
			foo := x.(string)
			fmt.Println(foo)
		}
		return
	}
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
	calculateBatteryPrice("10")
}



// special funtions as a support to make relatively short rules

func calculateBatteryPrice (hour string) {
	var flSoc float64
	var prices []float64
	var price string
	var flPrice float64
	soc := getItemState("Solarakku_SOC")
	flSoc,_ = strconv.ParseFloat(soc, 64) 
	flSoc -= 30
	hours := int(float64(flSoc / 8))
	intH,_ := strconv.Atoi(hour)
	for i := intH; i<24; i++ {
		price = getItemState(fmt.Sprintf("Tibber_total%02d",i))
		flPrice,_ = strconv.ParseFloat(price, 64)
		prices = append(prices, flPrice) 
	}
	sort.Float64s(prices)
	lPrices := len(prices)
	price = fmt.Sprintf("%0.4f",prices[lPrices-hours])
	fmt.Println("Bat-Price: ",price, hours)
	fmt.Println(prices)
	genVar.Pers.Set("!BAT_PRICE", price, cache.DefaultExpiration)
}
