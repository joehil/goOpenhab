package main

import (
	"fmt"
	"log"
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

	// Process current power
	if mInfo.Msgobject == "Tibber_Aktueller_Verbrauch" {
		var flInverter float64
		flNew, _ := strconv.ParseFloat(mInfo.Msgnewstate, 64)
		sEinAus := getItemState("Soyosource_EinAus")
		if sEinAus == "ON" {
			var inverter int
			strInverter := getItemState("Soyosource_Power_Value")
			debugLog(5, fmt.Sprint("Einlesen Inverter: ", strInverter))
			inverter, _ = strconv.Atoi(strInverter)
			flInverter = flNew * float64(0.5)
			inverter += int(flInverter)
			if inverter < 0 {
				inverter = 0
			}
			if inverter > 600 {
				inverter = 600
			}
			genVar.Pers.Set("Soyosource_Power_Value", fmt.Sprintf("%d", inverter), cache.DefaultExpiration)
			debugLog(5, fmt.Sprintf("Inverter: %d\n", inverter))
			// genVar.Mqttmsg <- Mqttparms{Topic: "inTopic", Message: fmt.Sprintf("%d", inverter)}
			genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Value: "state", Data: fmt.Sprintf("%d", inverter)}
		} else {
			lEinAus := getItemState("Laden_48_EinAus")
			if lEinAus == "ON" {
				var poti int
				digiPot := getItemState("Digipot_Poti")
				intDigiPot, _ := strconv.Atoi(digiPot)
				if intDigiPot > 240 && flNew < float64(-50) {
					// switch on laden_klein
					genVar.Postin <- Requestin{Node: "items", Item: "Steckdose_Jorg", Data: "ON"}
				}
				if intDigiPot < 80 && flNew > float64(0) {
					// switch off laden_klein
					genVar.Postin <- Requestin{Node: "items", Item: "Steckdose_Jorg", Data: "OFF"}
				}
				var flPoti float64 = flNew * float64(-0.255)
				poti = int(flPoti) + intDigiPot
				if poti > 255 {
					poti = 255
				}
				if poti < 0 {
					poti = 0
				}
				x, found := genVar.Pers.Get("!BATTERYLOAD")
				if found {
					if x == "1" {
						if poti < 127 {
							poti = 127
						}
					}
					if x == "2" {
						poti = 255
					}
				}
				if intDigiPot != poti {
					debugLog(5, fmt.Sprintf("Digipot setzen auf: %d", poti))
					// genVar.Mqttmsg <- Mqttparms{Topic: "digipot/inTopic", Message: fmt.Sprintf("%d", poti)}
					genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Value: "state", Data: fmt.Sprintf("%d", poti)}
					genVar.Pers.Set("Digipot_Poti", fmt.Sprintf("%d", poti), cache.DefaultExpiration)
				}
			}
		}
		return
	}

	// store every Tibber variable in cache
	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Tibber_t" || mInfo.Msgobject[0:8] == "Tibber_m" || mInfo.Msgobject[0:8] == "Tibber_n" {
			log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		}
	}

	// send message if ZOE is loaded less than 45%
	if len(mInfo.Msgobject) >= 26 {
		if mInfo.Msgobject == "Renault_Car_Batterieladung" && mInfo.Msgnewstate < "45.0" {
			log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Telegram <- "ZOE muss geladen werden (" + mInfo.Msgnewstate + "%)"
		}
	}

	// store the SOC of our battery in cache
	if mInfo.Msgobject == "Solarakku_SOC" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		log.Println("SOC stored: ", mInfo.Msgnewstate)
		return
	}

	// store the state of soyosource switch in cache
	if mInfo.Msgobject == "Soyosource_EinAus" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		log.Println("Soyosource_EinAus stored: ", mInfo.Msgnewstate)
		if mInfo.Msgnewstate == "OFF" {
			genVar.Mqttmsg <- Mqttparms{Topic: "inTopic", Message: "0"}
		}
		return
	}

	// store the state of load_48 switch in cache
	if mInfo.Msgobject == "Laden_48_EinAus" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		log.Println("Laden_48_EinAus stored: ", mInfo.Msgnewstate)
		return
	}

	// inform about sunrise, perform actions
	if (mInfo.Msgobject == "astro:sun:local:rise#event") &&
		(mInfo.Msgnewstate == "END") {
		guest := getItemState("gast_switch")
		if guest == "OFF" {
			genVar.Telegram <- "Sonnenaufgang"
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"OPEN\"}"}
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"OPEN\"}"}
			debugLog(3, "Open Rolladen Gaste Seite")
			debugLog(3, "Open Rolladen Gaste Vorne")
		}
		return
	}

	// inform about sunset, perform actions
	if (mInfo.Msgobject == "astro:sun:local:set#event") &&
		(mInfo.Msgnewstate == "END") {
		guest := getItemState("gast_switch")
		if guest == "OFF" {
			genVar.Telegram <- "Sonnenuntergang"
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"CLOSE\"}"}
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"CLOSE\"}"}
			debugLog(3, "Close Rolladen Gaste Seite")
			debugLog(3, "Close Rolladen Gaste Vorne")
		}
		return
	}

	// perform actions for several switches
	if mInfo.Msgobject == "Schalter_Rolladen_Gast_Action" {
		switch mInfo.Msgnewstate {
		case "1_single":
			// Rolladen Gast Seite open
			move := getItemState("Rolladen_Gast_Seite_Moving")
			if move == "STOP" {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"OPEN\"}"}
				log.Println("Rolladen Gast Seite open")
			} else {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"STOP\"}"}
				log.Println("Rolladen Gast Seite stop")
			}
		case "2_single":
			// Rolladen Gast Vorne open
			move := getItemState("Rolladen_Gast_vorne_Moving")
			if move == "STOP" {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"OPEN\"}"}
				log.Println("Rolladen Gast vorne open")
			} else {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"STOP\"}"}
				log.Println("Rolladen Gast vorne stop")
			}
		case "1_double":
			// Rolladen Gast Seite close
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"CLOSE\"}"}
			log.Println("Rolladen Gast Seite close")
		case "2_double":
			// Rolladen Gast Vorne close
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"CLOSE\"}"}
			log.Println("Rolladen Gast vorne close")
		default:
			return
		}
		genVar.Postin <- Requestin{Node: "items", Item: "Schalter_Rolladen_Gast_Action", Data: "reset"}
		return
	}

	// perform actions dimmer knob via MQTT
	dimmerKnob(mInfo, "WZDIMMER1", "action", "zigbee2mqtt/0xa4c1388f96c41f89", "zigbee2mqtt/0xf4b3b1fffef20459/l1/set")

	// perform actions dimmer knob via MQTT
	dimmerKnob(mInfo, "WZDIMMER2", "action", "zigbee2mqtt/0xa4c138672aa2c651", "zigbee2mqtt/0xf4b3b1fffef20459/l2/set")

	// perform actions for pushbutton via MQTT
	if mInfo.Msgobject == "zigbee2mqtt/0x00158d000893ac30" {
		switch readJson(mInfo.Msgnewstate, "action") {
		case "single":
		case "double":
		case "hold":
			// switch light on in our bedroom
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13855c794f2f9/set", Message: "{\"state\":\"TOGGLE\"}"}
			log.Println("Toggle bedroom light")
		default:
		}
		return
	}

	// MQTT pubhandler events
	//	if mInfo.Msgevent == "mqtt.pubhandler.event" {
	//		log.Println(mInfo.Msgevent, mInfo.Msgobject, readJson(mInfo.Msgnewstate, "action"))
	//	}

	// log internal events (restapi, mqtt, watchdog)
	if len(mInfo.Msgevent) >= 8 {
		if mInfo.Msgevent[0:7] == "restapi" || mInfo.Msgevent == "mqtt.reconnect.event" || mInfo.Msgevent == "watchdog.event" {
			log.Println(mInfo.Msgevent, mInfo.Msgobject)
			if mInfo.Msgevent == "watchdog.event" {
				restartNetwork()
				time.Sleep((5 * time.Second))
				panic("Watchdog called")
			}
		}
	}
}

func chronoEvents(mInfo Msginfo) {
	// this rule runs every minute
	var batPrice string
	var cmd string = "off"
	ap := getItemState("curr_price")
	soc := getItemState("Solarakku_SOC")
	stromGarage := getItemState("Balkonkraftwerk_Garage_Stromproduktion")
	stromBalkon := getItemState("Schalter_Balkon_Power")
	genVar.Getin <- Requestin{Node: "items", Item: "Laden_48_EinAus", Value: "state"}
	ladenSwitch := <-genVar.Getout
	as := getItemState("Tibber_Aktueller_Verbrauch")
	log.Printf("%s Current price: %s, SOC: %s, Strom Garage: %s, Strom Balkon %s, Aktueller Verbrauch %s, Laden-Ein/Aus: %s\n", mInfo.Msgobject, ap, soc, stromGarage, stromBalkon, as, ladenSwitch)
	flStromGarage, _ := strconv.ParseFloat(stromGarage, 64)
	flStromBalkon, _ := strconv.ParseFloat(stromBalkon, 64)
	flAs, _ := strconv.ParseFloat(as, 64)
	flAp, _ := strconv.ParseFloat(ap, 64)
	var flStrom float64 = flStromGarage + flStromBalkon
	x, found := genVar.Pers.Get("!BAT_PRICE")
	if found && flStrom < float64(100) && ladenSwitch == "OFF" {
		batPrice = x.(string)
		log.Println("BAT_PRICE: ", batPrice)
		flBatprice, _ := strconv.ParseFloat(batPrice, 64)
		if soc > "45.00" && flAp >= flBatprice {
			cmd = "unload"
		} else {
			cmd = "off"
		}
	}

	if (flStrom > float64(120) && ladenSwitch == "OFF" && flAs < float64(-50)) || (ladenSwitch == "ON" && flAs < float64(50)) ||
		flAp < float64(0.19) {
		cmd = "load"
	}
	if cmd == "off" {
		x, found := genVar.Pers.Get("!BATTERYLOAD")
		if found {
			if x == "1" {
				cmd = "load"
			}
			if x == "2" {
				cmd = "loadfull"
			}
		}
	}
	debugLog(5, fmt.Sprint("cmd: ", cmd))
	battery(cmd)

	// this rule runs at minute 2
	if strings.ContainsAny(mInfo.Msgobject[4:5], "2") {
		var btLoad string = "X"
		mt := getItemState("Tibber_mintotal")
		zone := getItemState("schalter_laden48_zone")
		debugLog(5, "Zone: Tibber_"+zone)
		zonePrice := getItemState("Tibber_" + zone)
		debugLog(5, "Zone price: "+zonePrice)
		//		ap := getItemState("curr_price")
		flMt, _ := strconv.ParseFloat(mt, 64)
		flCp, _ := strconv.ParseFloat(ap, 64)
		flZone, _ := strconv.ParseFloat(zonePrice, 64)
		debugLog(5, fmt.Sprintf("Zone price float: %0.4f", flZone))
		if soc < "44.00" && flMt >= flCp {
			btLoad = "1"
			log.Println("Battery Load on (emergency)")
		}
		if flZone >= flCp {
			btLoad = "2"
			log.Println("Battery Load on (zone)")
		}
		if (soc > "55.00" && btLoad == "1") || (flZone < flCp && btLoad == "2") {
			btLoad = "0"
			log.Println("Battery Load off")
		}
		genVar.Pers.Set("!BATTERYLOAD", btLoad, cache.DefaultExpiration)
	}

	// this rule runs at the first minute of each hour
	if mInfo.Msgobject[3:5] == "00" {
		setCurrentPrice(mInfo.Msgobject[0:2])
		calculateBatteryPrice(mInfo.Msgobject[0:2])
		return
	}

	// this rule runs at the second minute of each hour
	if mInfo.Msgobject[3:5] == "01" {
		doZoe := onOffByPrice(getItemState("schalter_zoe_zone"), mInfo.Msgobject)
		if doZoe {
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0x385b44fffe95ca3a/set", Message: "{\"state\":\"ON\"}"}
			log.Println("ZOE loading started")
		} else {
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0x385b44fffe95ca3a/set", Message: "{\"state\":\"OFF\"}"}
			log.Println("ZOE loading ended")
		}
		doPoessl := onOffByPrice("t3", mInfo.Msgobject)
		if doPoessl {
			genVar.Mqttmsg <- Mqttparms{Topic: "cmnd/tasmota_2EF5C7/POWER1", Message: "on"}
			log.Println("Poessl loading started")
		} else {
			genVar.Mqttmsg <- Mqttparms{Topic: "cmnd/tasmota_2EF5C7/POWER1", Message: "off"}
			log.Println("Poessl loading ended")
		}
		doWaschmaschine := onOffByPrice(getItemState("schalter_waschmaschine_zone"), mInfo.Msgobject)
		if doWaschmaschine {
			genVar.Mqttmsg <- Mqttparms{Topic: "cmnd/tasmota_68865C/POWER1", Message: "on"}
			log.Println("Waschmaschine on")
		} else {
			genVar.Mqttmsg <- Mqttparms{Topic: "cmnd/tasmota_68865C/POWER1", Message: "off"}
			log.Println("Waschmaschine off")
		}
		doBoiler := onOffByPrice("t4", mInfo.Msgobject)
		if doBoiler {
			genVar.Postin <- Requestin{Node: "items", Item: "shelly1pmWasserboiler1921680183_Betrieb", Data: "ON"}
			log.Println("Boiler on")
		} else {
			genVar.Postin <- Requestin{Node: "items", Item: "shelly1pmWasserboiler1921680183_Betrieb", Data: "OFF"}
			log.Println("Boiler off")
		}
		doLaden_klein := onOffByPrice("t2", mInfo.Msgobject)
		if doLaden_klein {
			genVar.Postin <- Requestin{Node: "items", Item: "Steckdose_Jorg_Betrieb", Value: "state", Data: "ON"}
			log.Println("Laden_klein on")
		} else {
			genVar.Postin <- Requestin{Node: "items", Item: "Steckdose_Jorg_Betrieb", Value: "state", Data: "OFF"}
			log.Println("Laden_klein off")
		}
		return
	}
}

// rules that are called when goOpenhab initializes

func rulesInit() {
	now := time.Now()
	hour := now.Hour()
	setCurrentPrice(fmt.Sprintf("%02d", hour))
	sEinAus := getItemState("Soyosource_EinAus")
	genVar.Pers.Set("Soyosource_EinAus", sEinAus, cache.DefaultExpiration)
	log.Println("Soyosource_EinAus stored: ", sEinAus)
	lEinAus := getItemState("Laden_48_EinAus")
	genVar.Pers.Set("Laden_48_EinAus", lEinAus, cache.DefaultExpiration)
	log.Println("Laden_48_EinAus stored: ", lEinAus)
	calculateBatteryPrice(fmt.Sprintf("%02d", hour))
}

// special funtions as a support to make relatively short rules

func calculateBatteryPrice(hour string) {
	var flSoc float64
	var flZone float64
	var prices []float64
	var price string
	var flPrice float64
	var hours int

	boolWeather := judgeWeather(4)
	soc := getItemState("Solarakku_SOC")
	zone := getItemState("Tibber_avg7")
	flZone, err := strconv.ParseFloat(zone, 64)
	if err != nil {
		flZone = float64(0.25)
	}
	flSoc, _ = strconv.ParseFloat(soc, 64)
	flSoc -= 50
	if flSoc < float64(0) {
		hours = 0
	} else {
		hours = int(float64((flSoc / 7)))
		hours += 1
	}
	intH, _ := strconv.Atoi(hour)
	if hour > "11" || (hour > "00" && boolWeather) {
		for i := intH; i < 24; i++ {
			price = getItemState(fmt.Sprintf("Tibber_total%02d", i))
			flPrice, _ = strconv.ParseFloat(price, 64)
			if flPrice > flZone {
				prices = append(prices, flPrice)
			}
		}
	}
	if hour <= "11" && hour > "00" && !boolWeather {
		for i := intH; i <= 11; i++ {
			price = getItemState(fmt.Sprintf("Tibber_total%02d", i))
			flPrice, _ = strconv.ParseFloat(price, 64)
			if flPrice > flZone {
				prices = append(prices, flPrice)
			}
		}
	}
	if hour == "00" {
		for i := intH; i <= 9; i++ {
			price = getItemState(fmt.Sprintf("Tibber_tomorrow%02d", i))
			flPrice, _ = strconv.ParseFloat(price, 64)
			if flPrice > flZone {
				prices = append(prices, flPrice)
			}
		}
	}
	if hour > "20" {
		for i := 0; i < 10; i++ {
			price = getItemState(fmt.Sprintf("Tibber_tomorrow%02d", i))
			flPrice, _ = strconv.ParseFloat(price, 64)
			if flPrice > flZone {
				prices = append(prices, flPrice)
			}
		}
	}
	sort.Float64s(prices)
	lPrices := len(prices)
	lPrices -= hours
	if lPrices < 0 {
		lPrices = 0
	}
	if len(prices) > 0 && lPrices < len(prices) {
		price = fmt.Sprintf("%0.4f", prices[lPrices])
	} else {
		price = "9.9999"
	}
	log.Println("Bat-Price: ", price, hours)
	log.Println(prices)
	genVar.Pers.Set("!BAT_PRICE", price, cache.DefaultExpiration)
}

func battery(cmd string) {
	switch cmd {
	case "off":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.DefaultExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.DefaultExpiration)
	case "load":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.DefaultExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Laden_48 switched on")
	case "loadfull":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.DefaultExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Laden_48 switched on full")
	case "unload":
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.DefaultExpiration)
		// Soyosource on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Soyosource switched on")
	default:
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.DefaultExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.DefaultExpiration)
	}
}

func onOffByPrice(zone string, obj string) bool {
	var flPrice float64 = 0
	var flCurr float64 = 0
	var hour string = obj[0:2]
	var err error
	if !(hour >= "21" || hour <= "06") && zone[0:1] == "n" {
		return false
	}
	price := getItemState(fmt.Sprintf("Tibber_%s", zone))
	flPrice, err = strconv.ParseFloat(price, 64)
	if err != nil {
		log.Println("Price was not found", err)
		return false
	}
	curr_price := getItemState("curr_price")
	flCurr, err = strconv.ParseFloat(curr_price, 64)
	if err != nil {
		log.Println("Current price was not found", err)
		return false
	}
	return flCurr <= flPrice
}

func judgeWeather(search int) bool {
	var result bool = false
	genVar.Getin <- Requestin{Node: "items", Item: "Weather_Information_Condition", Value: "state"}
	weather := <-genVar.Getout
	intWeather, err := strconv.Atoi(weather)
	if err == nil {
		result = intWeather >= search
	}
	return result
}

func setCurrentPrice(h string) {
	item := "Tibber_total" + h
	if h == "00" {
		item = "Tibber_tomorrow00"
	}
	log.Println(item)
	genVar.Getin <- Requestin{Node: "items", Item: item, Value: "state"}
	answer := <-genVar.Getout
	log.Println(answer)
	if answer != "" {
		genVar.Postin <- Requestin{Node: "items", Item: "curr_price", Value: "state", Data: answer}
	}
}
