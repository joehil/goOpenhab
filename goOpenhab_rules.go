package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/tidwall/gjson"
)

func processRulesInfo(mInfo Msginfo) {
	msgLog(mInfo)

	atime := time.Now()
	timeVar.hour = atime.Hour()
	timeVar.minute = atime.Minute()
	timeVar.second = atime.Second()
	timeVar.day = atime.Day()
	timeVar.month = atime.Month()
	timeVar.year = atime.Year()
	timeVar.day = atime.Day()
	timeVar.yearday = atime.YearDay()
	timeVar.weekday = atime.Weekday()
	timeVar.dayminute = timeVar.hour*60 + timeVar.minute

	debugLog(9, fmt.Sprintf("%s;%s;%s;%s;%s", mInfo.Msgevent, mInfo.Msgobjtype, mInfo.Msgobject, mInfo.Msgoldstate, mInfo.Msgnewstate))
	if mInfo.Msgevent == "chrono.event" {
		chronoEvents(mInfo)
		return
	}

	if mInfo.Msgobject == "Tibber_Aktueller_Preis" {
		genVar.Postin <- Requestin{Node: "items", Item: "curr_price", Data: mInfo.Msgnewstate}
	}

	// Process current power
	if mInfo.Msgobject == "Tibber_Aktueller_Verbrauch" {
		var flInverter float64
		flNew, _ := strconv.ParseFloat(mInfo.Msgnewstate, 64)
		sEinAus := getItemState("Soyosource_EinAus")
		if flNew > 1400 && sEinAus == "ON" {
			sEinAus = "OFF"
			// Soyosource off
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
			log.Println("Soyosource switched off")
			genVar.Pers.Set("Soyosource_Power_Value", "0", cache.NoExpiration)
			genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		}
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
			genVar.Pers.Set("Soyosource_Power_Value", fmt.Sprintf("%d", inverter), cache.NoExpiration)
			debugLog(5, fmt.Sprintf("Inverter: %d\n", inverter))
			// genVar.Mqttmsg <- Mqttparms{Topic: "inTopic", Message: fmt.Sprintf("%d", inverter)}
			genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Value: "state", Data: fmt.Sprintf("%d", inverter)}
		} else {
			lEinAus := getItemState("Laden_48_EinAus")
			if lEinAus == "ON" {
				var poti int
				//	soc := getItemState("Solarakku_SOC")
				soc := getSOCstr()
				digiPot := getItemState("Digipot_Poti")
				debugLog(5, "String digipot: "+digiPot)
				debugLog(5, "String SOC: "+soc)
				intDigiPot, _ := strconv.Atoi(digiPot)
				x, found := genVar.Pers.Get("!LADEN_KLEIN")
				if (intDigiPot > 240 && flNew < float64(-50)) || (found && x == "ON") || (soc >= "96" && flNew < float64(-50)) {
					// switch on laden_klein
					genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_laden_klein", Data: "ON"}
				}
				if x != "ON" || !found {
					if intDigiPot < 80 && flNew > float64(0) && soc < "96" {
						// switch off laden_klein
						genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_laden_klein", Data: "OFF"}
					}
				}
				var flPoti float64 = flNew * float64(-0.255)
				poti = int(flPoti) + intDigiPot
				if poti > 255 {
					poti = 255
				}
				if poti < 0 {
					poti = 0
				}
				x, found = genVar.Pers.Get("!BATTERYLOAD")
				if found {
					debugLog(5, fmt.Sprintf("Batteryload: %s", x))
					if x == "1" {
						if poti < 127 {
							poti = 127
						}
					}
					if x == "2" {
						poti = 255
					}
				}
				if soc >= "96" {
					poti = 0
					battery("off")
				}
				debugLog(5, fmt.Sprintf("Digipot old: %d new %d flNew %0.2f", intDigiPot, poti, flNew))
				tmint := time.Now().Second() % 10
				if tmint == 0 || poti != intDigiPot {
					debugLog(5, fmt.Sprintf("Digipot setzen auf: %d", poti))
					// genVar.Mqttmsg <- Mqttparms{Topic: "digipot/inTopic", Message: fmt.Sprintf("%d", poti)}
					genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: fmt.Sprintf("%d", poti)}
					genVar.Pers.Set("Digipot_Poti", fmt.Sprintf("%d", poti), cache.NoExpiration)
				}
			} else {
				genVar.Postin <- Requestin{Node: "items", Item: "Steckdose_Jorg", Data: "OFF"}
			}
		}
		return
	}

	// store every Tibber variable in cache
	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Tibber_t" || mInfo.Msgobject[0:8] == "Tibber_m" || mInfo.Msgobject[0:8] == "Tibber_n" {
			log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.NoExpiration)
		}
	}

	// send message if ZOE is loaded less than 45%
	if len(mInfo.Msgobject) >= 26 {
		if mInfo.Msgobject == "Renault_Car_Batterieladung" {
			plug := getItemState("Renault_Car_Plug_Status")
			sw := getItemState("Schalter_ZOE_EinAus")
			if strings.ToUpper(sw) == "OFF" || strings.ToUpper(plug) != "PLUGGED" {
				if mInfo.Msgnewstate < "45.0" {
					log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
					genVar.Telegram <- "ZOE muss geladen werden (" + mInfo.Msgnewstate + "%)"
				}
			}
		}
	}

	// react to inverter temperature
	if mInfo.Msgobject == "Balkonkraftwerk_Garage_balkonkraft_garage_temp" {
		onOff := getItemState("garage_ventilator_garage_ventilator_onoff")
		fltemp, _ := strconv.ParseFloat(mInfo.Msgnewstate, 64)
		debugLog(5, fmt.Sprintf("Balkonkraftwerk_Garage_balkonkraft_garage_temp: %.1f", fltemp))
		if fltemp > float64(38) && onOff == "OFF" {
			genVar.Postin <- Requestin{Node: "items", Item: "garage_ventilator_garage_ventilator_onoff", Data: "ON"}
			debugLog(5, "Garage_Ventilator on")
		}
		if fltemp < float64(30) && onOff == "ON" {
			genVar.Postin <- Requestin{Node: "items", Item: "garage_ventilator_garage_ventilator_onoff", Data: "OFF"}
			debugLog(5, "Garage_Ventilator off")
		}
		return
	}

	// store the SOC of our battery in cache
	if mInfo.Msgobject == "Solarakku_SOC" || mInfo.Msgobject == "battery_can_SOC" {
		genVar.Pers.Set("SOC", mInfo.Msgnewstate, cache.NoExpiration)
		log.Println("SOC stored: ", mInfo.Msgnewstate)
		return
	}

	// store the state of soyosource switch in cache
	if mInfo.Msgobject == "Soyosource_EinAus" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.NoExpiration)
		log.Println("Soyosource_EinAus stored: ", mInfo.Msgnewstate)
		if mInfo.Msgnewstate == "OFF" {
			genVar.Mqttmsg <- Mqttparms{Topic: "inTopic", Message: "0"}
		}
		return
	}

	// store the state of load_48 switch in cache
	if mInfo.Msgobject == "Laden_48_EinAus" {
		genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.NoExpiration)
		log.Println("Laden_48_EinAus stored: ", mInfo.Msgnewstate)
		return
	}

	// store pv forecast in item
	if mInfo.Msgevent == "pvforecast.watthours.event" {
		genVar.Pers.Set("pv_forecast_"+mInfo.Msgobject, mInfo.Msgnewstate, cache.NoExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "pv_forecast_" + mInfo.Msgobject, Data: mInfo.Msgnewstate}
		log.Println("pv_forecast_"+mInfo.Msgobject, mInfo.Msgnewstate)
		return
	}

	// inform about sunrise, perform actions
	if (mInfo.Msgobject == "astro:sun:local:rise#event") &&
		(mInfo.Msgnewstate == "END") {
		guest := getItemState("gast_switch")
		if guest == "OFF" {
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"OPEN\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"OPEN\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138f57159f18d/set", Message: "{\"state\":\"OPEN\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13887c35e3920/set", Message: "{\"state\":\"OPEN\"}"}
			debugLog(3, "Open Rolladen Gaste Seite")
			debugLog(3, "Open Rolladen Gaste Vorne")
			debugLog(3, "Open Rolladen Joerg")
			debugLog(3, "Open Rolladen Buero")
		}
		genVar.Telegram <- "Sonnenaufgang"
		return
	}

	// inform about sunset, perform actions
	if (mInfo.Msgobject == "astro:sun:local:set#event") &&
		(mInfo.Msgnewstate == "END") {
		guest := getItemState("gast_switch")
		if guest == "OFF" {
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138bac3fa8036/set", Message: "{\"state\":\"CLOSE\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c1384bce7c2ebb/set", Message: "{\"state\":\"CLOSE\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138f57159f18d/set", Message: "{\"state\":\"CLOSE\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138a53a4a83d4/set", Message: "{\"state\":\"CLOSE\"}"}
			time.Sleep(20 * time.Second)
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13887c35e3920/set", Message: "{\"state\":\"CLOSE\"}"}
			debugLog(3, "Close Rolladen Gaste Seite")
			debugLog(3, "Close Rolladen Gaste Vorne")
			debugLog(3, "Close Rolladen Joerg")
			debugLog(3, "Close Rolladen Buero")
			debugLog(3, "Close Rolladen Bad")
		}
		genVar.Telegram <- "Sonnenuntergang"
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

	if mInfo.Msgobject == "Schalter_Rolladen_Bad_schalter_rolladen_bad_action" {
		switch mInfo.Msgnewstate {
		case "single":
			// Rolladen Bad Seite open
			move := getItemState("Rolladen_Bad_rolladen_bad_moving")
			if move == "STOP" {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138a53a4a83d4/set", Message: "{\"state\":\"OPEN\"}"}
				log.Println("Rolladen Bad open")
			} else {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138a53a4a83d4/set", Message: "{\"state\":\"STOP\"}"}
				log.Println("Rolladen Bad stop")
			}
		case "double":
			// Rolladen Bad close
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138a53a4a83d4/set", Message: "{\"state\":\"CLOSE\"}"}
			log.Println("Rolladen Bad close")
		default:
			return
		}
		genVar.Postin <- Requestin{Node: "items", Item: "Schalter_Rolladen_Bad_schalter_rolladen_bad_action", Data: "reset"}
		return
	}

	// perform actions for rolladen Joerg via MQTT
	if mInfo.Msgobject == "Schalter_Rollagen_Joerg_schalter_rolladen_joerg_action" {
		switch mInfo.Msgnewstate {
		case "single":
			// Rolladen Joerg Seite open
			move := getItemState("Rolladen_Joerg_rolladen_joerg_moving")
			if move == "STOP" {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138f57159f18d/set", Message: "{\"state\":\"OPEN\"}"}
				log.Println("Rolladen Joerg open")
			} else {
				genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138f57159f18d/set", Message: "{\"state\":\"STOP\"}"}
				log.Println("Rolladen Joerg stop")
			}
		case "double":
			// Rolladen Joerg close
			genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138f57159f18d/set", Message: "{\"state\":\"CLOSE\"}"}
			log.Println("Rolladen Joerg close")
		default:
			return
		}
		genVar.Postin <- Requestin{Node: "items", Item: "Schalter_Rolladen_Joerg_schalter_rolladen_joerg_action", Data: "reset"}
		return
	}

	// perform actions dimmer knob via MQTT
	dimmerKnob(mInfo, "WZDIMMER1", "action", "zigbee2mqtt/0xa4c1388f96c41f89", "zigbee2mqtt/0xf4b3b1fffef20459/l1/set")

	// perform actions dimmer knob via MQTT
	dimmerKnob(mInfo, "WZDIMMER2", "action", "zigbee2mqtt/0xa4c138672aa2c651", "zigbee2mqtt/0xf4b3b1fffef20459/l2/set")

	// perform actions for pushbutton Joerg via MQTT
	if mInfo.Msgobject == "zigbee2mqtt/0x00158d000893ac30" {
		action := readJson(mInfo.Msgnewstate, "action")
		log.Println("Pushbutton Joerg: ", action)
		switch action {
		case "single":
			itemToggle("Licht_Zigbee_licht_flur_oben_onoff")
		case "double":
			itemToggle("Licht_Zigbee_licht_flur_eg_onoff")
		case "hold":
			itemToggle("Schalter_Schlafzimmer_EinAus")
		default:
		}
		return
	}

	// perform actions for pushbutton Brigitte via MQTT
	if mInfo.Msgobject == "zigbee2mqtt/0x00158d0007c0cbf2" {
		action := readJson(mInfo.Msgnewstate, "action")
		log.Println("Pushbutton Brigitte: ", action)
		switch action {
		case "single":
			itemToggle("Licht_Zigbee_licht_flur_oben_onoff")
		case "double":
		case "hold":
			itemToggle("Schalter_Schlafzimmer_EinAus")
		default:
		}
		return
	}

	// perform actions for pushbutton Flur_EG via MQTT
	if mInfo.Msgobject == "zigbee2mqtt/0x187a3efffe0f5a35" {
		switch readJson(mInfo.Msgnewstate, "action") {
		case "1_single":
			itemToggle("Licht_Zigbee_licht_flur_oben_onoff")
		case "2_single":
			itemToggle("Licht_Zigbee_licht_flur_eg_onoff")
		case "3_single":
			itemToggle("Wandschrank_1_EinAus")
		case "4_single":
			itemToggle("Wandschrank_2_EinAus")
		default:
		}
		return
	}

	// perform actions for pushbutton Kellertür via MQTT
	if mInfo.Msgobject == "zigbee2mqtt/0x00158d0007c0d18a" {
		switch readJson(mInfo.Msgnewstate, "action") {
		case "single":
			genVar.Postin <- Requestin{Node: "items", Item: "Licht_Zigbee_licht_flur_keller_onoff", Data: "ON"}
		default:
		}
		return
	}

	// MQTT doorlock events
	if len(mInfo.Msgobject) >= 12 {
		if mInfo.Msgobject[0:9] == "doorlock/" && mInfo.Msgobject[0:12] != "doorlock/in/" {
			log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
		}
		if len(mInfo.Msgnewstate) >= 16 {
			if mInfo.Msgobject == "doorlock/message" && mInfo.Msgnewstate[0:16] == "TAG: door opened" {
				recordVideo("http://192.168.0.168:81/stream", "15")
				genVar.Telegram <- mInfo.Msgnewstate
				return
			}
		}
	}

	// MQTT LoRa events
	if len(mInfo.Msgobject) >= 18 {
		if mInfo.Msgevent == "mqtt.pubhandler.event" &&
			mInfo.Msgobject[0:18] == "LoRa2MQTT/outTopic" {
			//			log.Println(mInfo.Msgevent, mInfo.Msgobject, mInfo.Msgnewstate)
			erg := strings.Split(mInfo.Msgobject, "/")
			if erg[2] == "200" && erg[3] == "11" {
				ttemp, err := strconv.ParseFloat(mInfo.Msgnewstate, 64)
				if err == nil {
					stemp := fmt.Sprintf("%.2f", ttemp/100)
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_oben_temperatur", Data: stemp}
				}
			}
			if erg[2] == "200" && erg[3] == "80" {
				var hOben string = "IIIII"
				x, found := genVar.Pers.Get("!HEIZUNG_OBEN")
				if found {
					hOben = x.(string)
				}
				if mInfo.Msgnewstate[0:5] != hOben {
					genVar.Mqttmsg <- Mqttparms{Topic: "LoRa2MQTT/inTopic/200/80/1", Message: hOben}
				}
				if mInfo.Msgnewstate[0:1] == "N" {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_bad", Data: "ON"}
				} else {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_bad", Data: "OFF"}
				}
				if mInfo.Msgnewstate[1:2] == "N" {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_gaestezimmer", Data: "ON"}
				} else {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_gaestezimmer", Data: "OFF"}
				}
				if mInfo.Msgnewstate[2:3] == "N" {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_brigitte", Data: "ON"}
				} else {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_brigitte", Data: "OFF"}
				}
				if mInfo.Msgnewstate[3:4] == "N" {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_schlafzimmer", Data: "ON"}
				} else {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_schlafzimmer", Data: "OFF"}
				}
				if mInfo.Msgnewstate[4:5] == "N" {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_joerg", Data: "ON"}
				} else {
					genVar.Postin <- Requestin{Node: "items", Item: "heizung_joerg", Data: "OFF"}
				}
			}
		}
	}

	if mInfo.Msgobject == "Thermometer_Bad_TemperatureZusatz" || mInfo.Msgevent == "periodic15.event" {
		time.Sleep(2 * time.Second)
		setHeating("ZBMultiHeatingSwitch_Oben_ZBMultiHeatingSwitch_Oben_Bad", "Soll_Temperatur_Bad", "Thermometer_Bad_TemperatureZusatz")
	}
	if mInfo.Msgobject == "Thermometer_Buero_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Oben_ZBMultiHeatingSwitch_Oben_Brigitte", "Soll_Temperatur_Buero", "Thermometer_Buero_Temperature")
	}
	if mInfo.Msgobject == "ObergeschossThermometer_Guest_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Oben_ZBMultiHeatingSwitch_Oben_Gast", "Soll_Temperatur_Gast", "ObergeschossThermometer_Guest_Temperature")
	}
	if mInfo.Msgobject == "Thermometer_Jorg_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Oben_ZBMultiHeatingSwitch_Oben_Joerg", "Soll_Temperatur_Joerg", "Thermometer_Jorg_Temperature")
	}
	if mInfo.Msgobject == "Thermometer_Schlafzimmer_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Oben_ZBMultiHeatingSwitch_Oben_Schlafzimmer", "Soll_Temperatur_Schlafzimmer", "Thermometer_Schlafzimmer_Temperature")
	}
	if mInfo.Msgobject == "Temperatur_Wohnzimmer_Temperatur_Wohnzimmer_Wert" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Unten_ZBMultiHeatingSwitch_Unten_Wohnzimmer1", "Soll_Temperatur_Wohnzimmer", "Temperatur_Wohnzimmer_Temperatur_Wohnzimmer_Wert")
		setHeating("ZBMultiHeatingSwitch_Unten_ZBMultiHeatingSwitch_Unten_Wohnzimmer2", "Soll_Temperatur_Wohnzimmer", "Temperatur_Wohnzimmer_Temperatur_Wohnzimmer_Wert")
	}
	if mInfo.Msgobject == "Thermometer_WC_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Unten_ZBMultiHeatingSwitch_Unten_Gaesteklo", "Soll_Temperatur_Gaesteklo", "Thermometer_WC_Temperature")
	}
	if mInfo.Msgobject == "Thermometer_Esszimmer_Temperature" || mInfo.Msgevent == "periodic15.event" {
		setHeating("ZBMultiHeatingSwitch_Unten_ZBMultiHeatingSwitch_Unten_Esszimmer", "Soll_Temperatur_Esszimmer", "Thermometer_Esszimmer_Temperature")
	}

	if mInfo.Msgobject == "Thermometer_Bad_Luftfeuchtigkeit" {
		log.Println("Bad Luftfeuchtigkeit:", mInfo.Msgnewstate)
		if mInfo.Msgnewstate > "60" {
			log.Println("Bad Luftfeuchtigkeit 60% - Schalte Lüfter an")
			genVar.Postin <- Requestin{Node: "items", Item: "Luefter_Bad_luefter_bad_onoff", Data: "ON"}
		}
		if mInfo.Msgnewstate < "55" {
			log.Println("Bad Luftfeuchtigkeit 55% - Schalte Lüfter aus")
			genVar.Postin <- Requestin{Node: "items", Item: "Luefter_Bad_luefter_bad_onoff", Data: "OFF"}
		}
	}

	// log internal events (restapi, mqtt, watchdog)
	if len(mInfo.Msgevent) >= 8 {
		if mInfo.Msgevent[0:7] == "restapi" || mInfo.Msgevent == "mqtt.reconnect.event" || mInfo.Msgevent == "watchdog.event" {
			log.Println(mInfo.Msgevent, mInfo.Msgobject)
			if mInfo.Msgevent == "watchdog.event" && rules_active {
				filename := "/tmp/goOpenhab_watchdog.semaphore"
				_, err := os.Stat(filename)
				if err == nil {
					log.Println("Rebooting system")
					//					genVar.Telegram <- "Rebooting system"
					reboot()
					return
				}
				log.Println("Restart network")
				//				genVar.Telegram <- "Restart network"
				_, err = os.Create(filename)
				log.Println("File error:", err)
				if err != nil {
					log.Fatal(err)
				} else {
					log.Println("Semaphore file created: ", filename)
				}
				restartNetwork()
				time.Sleep((5 * time.Second))
				panic("Restart network")
			}
		}
	}

	if (mInfo.Msgobject == "Bewegungsmelder_1_EinAus" || mInfo.Msgobject == "Bewegungsmelder_2_EinAus") && mInfo.Msgnewstate == "ON" {
		debugLog(5, "Bewegungsmelder Flur oben")
		genVar.Postin <- Requestin{Node: "items", Item: "Licht_Zigbee_licht_flur_oben_onoff", Data: "ON"}
		setItemOffTime("Licht_Zigbee_licht_flur_oben_onoff", 300)
		return
	}

	if mInfo.Msgobject == "Bewegungsmelder_3_EinAus" && mInfo.Msgnewstate == "ON" {
		var pac float64 = 0
		var guest string = "OFF"
		debugLog(5, "Bewegungsmelder Flur oben und EG")
		genVar.Postin <- Requestin{Node: "items", Item: "Licht_Zigbee_licht_flur_oben_onoff", Data: "ON"}
		setItemOffTime("Licht_Zigbee_licht_flur_oben_onoff", 300)
		x, found := genVar.Pers.Get("!BalkonPAC")
		if found {
			flPac, err := strconv.ParseFloat(x.(string), 64)
			if err == nil {
				pac = flPac
			}
		}
		x, found = genVar.Pers.Get("!GUEST")
		if found {
			guest = x.(string)
		}
		if pac < float64(50) || guest == "ON" {
			genVar.Postin <- Requestin{Node: "items", Item: "Licht_Zigbee_licht_flur_eg_onoff", Data: "ON"}
		}
		return
	}

	if mInfo.Msgevent == "network.availability.machine.event" && mInfo.Msgnewstate == "999" && rules_active {
		log.Println(mInfo.Msgevent, mInfo.Msgobject, mInfo.Msgnewstate)
		reboot()
		time.Sleep((5 * time.Second))
		panic("Reboot started")
	}

	if mInfo.Msgevent == "network.availability.internet.event" && mInfo.Msgnewstate == "999" {
		log.Println(mInfo.Msgevent, mInfo.Msgobject, mInfo.Msgnewstate)
		exec_cmd("/opt/homeautomation/fritzbox_reboot.sh")
		return
	}

	if mInfo.Msgobject == "Network_Device_19216801_Pingzeit" {
		log.Println(mInfo.Msgevent, mInfo.Msgobject, mInfo.Msgnewstate)
		//	exec_cmd("/opt/homeautomation/fritzbox_reboot.sh")
		return
	}

	if mInfo.Msgobject == "Lichtschalter_Buro_Jorg" {
		if mInfo.Msgnewstate == "ON" {
			setItemOffTime(mInfo.Msgobject, 300)
		} else {
			setItemOffTime(mInfo.Msgobject, 0)
		}
		return
	}

	if mInfo.Msgobject == "Licht_Zigbee_licht_flur_oben_onoff" {
		if mInfo.Msgnewstate == "ON" {
			setItemOffTime(mInfo.Msgobject, 300)
		} else {
			setItemOffTime(mInfo.Msgobject, 0)
		}
		return
	}

	if mInfo.Msgobject == "Licht_Zigbee_licht_flur_eg_onoff" {
		if mInfo.Msgnewstate == "ON" {
			setItemOffTime(mInfo.Msgobject, 300)
		} else {
			setItemOffTime(mInfo.Msgobject, 0)
		}
		return
	}

	if mInfo.Msgobject == "Licht_Zigbee_licht_flur_keller_onoff" {
		if mInfo.Msgnewstate == "ON" {
			setItemOffTime(mInfo.Msgobject, 1200)
		} else {
			setItemOffTime(mInfo.Msgobject, 0)
		}
		return
	}

	if mInfo.Msgobject == "Zigbee_Steckdosen_steckdose_inverter_klein_power" {
		flPower, _ := strconv.ParseFloat(mInfo.Msgnewstate, 64)
		if flPower > 90 && flPower < 100 {
			genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_inverter_klein", Data: "OFF"}
			log.Println(mInfo.Msgobject, mInfo.Msgnewstate)
		}
		return
	}

	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Heizung_" {
			debugLog(7, "Alarm set for "+mInfo.Msgobject)
			setItemAlarmTime(mInfo.Msgobject, 900)
			return
		}
	}

	if mInfo.Msgobject == "desiredState/in" {
		log.Println(mInfo.Msgobject + " : " + mInfo.Msgnewstate)
		vrs := strings.Split(mInfo.Msgnewstate, ":")
		if len(vrs) == 1 {
			vrs[1] = ""
		}
		log.Println(vrs)
		putState(vrs[0], vrs[1])
		return
	}

	if mInfo.Msgobject == "check_joehil_Last_Failure" {
		log.Println(mInfo.Msgobject + " : " + mInfo.Msgnewstate)
		genVar.Telegram <- "Verbindung zu joehil.de verloren"
		return
	}

	if mInfo.Msgobject == "ZigbeeCheck_zigbeecheckjson" {
		var summe float64 = 0
		var messages float64 = 0
		v1 := gjson.Get(mInfo.Msgnewstate, "os.memory_percent")
		log.Println("Zigbee memory percent: ", v1)
		result := gjson.Get(mInfo.Msgnewstate, "devices")
		result.ForEach(func(key, value gjson.Result) bool {
			cnt1 := gjson.Get(value.String(), "leave_count")
			cnt2 := gjson.Get(value.String(), "messages")
			summe += cnt1.Num
			messages += cnt2.Num
			return true // keep iterating
		})
		log.Printf("Zigbee Summe Leaves: %.2f  Summe Messages: %.2f\n", summe, messages)
	}
}

func chronoEvents(mInfo Msginfo) {
	// this rule runs every minute
	var batPrice string
	var cmd string = "off"
	ap := getItemState("curr_price")
	// soc := getItemState("Solarakku_SOC")
	soc := getSOCstr()
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
	if found && flStrom < float64(150) && ladenSwitch == "OFF" {
		batPrice = x.(string)
		log.Println("BAT_PRICE: ", batPrice)
		flBatprice, _ := strconv.ParseFloat(batPrice, 64)
		if (soc > "22.00" || soc == "100") && flAp >= flBatprice {
			if flAs > float64(1400) && soc < "55" {
				cmd = "off"
			} else {
				cmd = "unload"
			}
		} else {
			cmd = "off"
		}
	}

	if (flStrom > float64(120) && ladenSwitch == "OFF" && flAs < float64(-50)) ||
		(ladenSwitch == "ON" && flAs < float64(50)) ||
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

	go iterateOffs()
	go iterateAlarms()

	// emergency switch off
	if mInfo.Msgobject == "00:00" || mInfo.Msgobject == "01:00" || mInfo.Msgobject == "02:00" {
		log.Println("Switch lights off in living room")
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13874b6060fe9/set", Message: "{\"state_l1\":\"OFF\"}"}
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13874b6060fe9/set", Message: "{\"state_l2\":\"OFF\"}"}
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c13843caca9572/set", Message: "{\"state\":\"OFF\"}"}
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138c1f0eacf1d/set", Message: "{\"state\":\"OFF\"}"}
	}

	// reboot fritzbox every 2 days at 03.17
	/*	if mInfo.Msgobject == "03:17" {
		d := time.Now()
		day := d.Day()
		if day%2 == 0 {
			log.Println("Reboot Fritzbox")
			exec_cmd("/opt/homeautomation/fritzbox_reboot.sh")
		}
	} */

	// this rule runs at minutes ending at 2 and 7
	if strings.ContainsAny(mInfo.Msgobject[4:5], "27") {
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
		debugLog(1, fmt.Sprintf("Zone price float: %0.4f", flZone))

		x, found := genVar.Pers.Get("!BATTERYLOAD")
		if found {
			btLoad = x.(string)
			debugLog(1, "!BATTERYLOAD: "+btLoad)
		}

		if soc < "21.00" && soc != "100" && flMt >= flCp {
			btLoad = "1"
			log.Println("Battery Load on (emergency)")
		}
		if onOffByPrice(zone, mInfo.Msgobject) {
			btLoad = "2"
			log.Println("Battery Load on (zone)")
		}
		if (soc > "28.00" && btLoad == "1") || (flZone < flCp && btLoad == "2") {
			btLoad = "0"
			log.Println("Battery Load off")
		}
		debugLog(1, "BtLoad: "+btLoad)
		if btLoad != "X" {
			genVar.Pers.Set("!BATTERYLOAD", btLoad, cache.NoExpiration)
		}
		return
	}

	// this rule runs at the first minute of each hour
	if (mInfo.Msgobject[3:5] == "00" || mInfo.Msgobject[3:5] == "30" || mInfo.Msgobject[3:5] == "15" || mInfo.Msgobject[3:5] == "45") &&
		mInfo.Msgobject != "00:00" || mInfo.Msgobject == "00:05" {
		//	setCurrentPrice(mInfo.Msgobject[0:2])
		time.Sleep(15 * time.Second)
		calculateBatteryPrice(mInfo.Msgobject[0:2])
		log.Println(getWeather())
		genVar.Postin <- Requestin{Node: "items", Item: "meteomatics_weather", Data: getWeather()}

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
		//doBoiler := onOffByPrice("t4", mInfo.Msgobject)
		if doBoiler {
			genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_heisswasser_onoff", Data: "ON"}
			log.Println("Boiler on")
		} else {
			genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_heisswasser_onoff", Data: "OFF"}
			log.Println("Boiler off")
		}
		doLaden_klein := onOffByPrice("mintotal", mInfo.Msgobject)
		if doLaden_klein {
			genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_laden_klein", Value: "state", Data: "ON"}
			genVar.Pers.Set("!LADEN_KLEIN", "ON", cache.NoExpiration)
			log.Println("Laden_klein on")
		} else {
			genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_laden_klein", Value: "state", Data: "OFF"}
			genVar.Pers.Delete("!LADEN_KLEIN")
			log.Println("Laden_klein off")
		}
		pac := getItemState("Balkonkraftwerk_Garage_Stromproduktion")
		genVar.Pers.Set("!BalkonPAC", pac, cache.NoExpiration)
		guest := getItemState("gast_switch")
		genVar.Pers.Set("!GUEST", guest, cache.NoExpiration)

		return
	}

	// this rule runs at minutes ending at 1
	if strings.ContainsAny(mInfo.Msgobject[4:5], "1") {
		gast := getItemState("FHEM_Gast_da")
		log.Println("Gast:", gast[0:2])
		if gast[0:2] == "ja" {
			log.Println("Im Gast")
			heiz := getItemState("FHEM_Heizung_Julia")
			log.Println("Heizung:", heiz[0:2])
			if heiz[0:2] == "on" {
				genVar.Postin <- Requestin{Node: "items", Item: "Hilfsheizung_Hilfsheizung_Gaestezimmer", Value: "state", Data: "ON"}
			} else {
				genVar.Postin <- Requestin{Node: "items", Item: "Hilfsheizung_Hilfsheizung_Gaestezimmer", Value: "state", Data: "OFF"}
			}
		} else {
			genVar.Postin <- Requestin{Node: "items", Item: "Hilfsheizung_Hilfsheizung_Gaestezimmer", Value: "state", Data: "OFF"}
		}
	}
}

// rules that are called when goOpenhab initializes

func rulesInit() int {
	uptime, _ := getSystemUptime()
	log.Printf("Uptime: %f\n", uptime)
	if uptime < float64(300) {
		log.Println("Waiting for system to be ready")
		time.Sleep(time.Second * 300)
		log.Println("Reinitializing goOpenhab rules...")
		return 99
	}
	now := time.Now()
	hour := now.Hour()
	exec_cmd("/opt/homeautomation/tibber2mqtt", "reinit")
	time.Sleep(time.Second)
	setCurrentPrice(fmt.Sprintf("%02d", hour))
	sEinAus := getItemState("Soyosource_EinAus")
	genVar.Pers.Set("Soyosource_EinAus", sEinAus, cache.NoExpiration)
	log.Println("Soyosource_EinAus stored: ", sEinAus)
	lEinAus := getItemState("Laden_48_EinAus")
	genVar.Pers.Set("Laden_48_EinAus", lEinAus, cache.NoExpiration)
	log.Println("Laden_48_EinAus stored: ", lEinAus)
	calculateBatteryPrice(fmt.Sprintf("%02d", hour))
	pac := getItemState("Balkonkraftwerk_Garage_Stromproduktion")
	genVar.Pers.Set("!BalkonPAC", pac, cache.NoExpiration)
	log.Println("BalkonPAC stored: ", pac)
	guest := getItemState("gast_switch")
	genVar.Pers.Set("!GUEST", guest, cache.NoExpiration)
	log.Println("Guest stored: ", guest)
	genVar.Pers.Set("!HEIZUNG_OBEN", "NNNNN", cache.NoExpiration)

	tZoe_zone := getItemState("schalter_zoe_zone")
	if tZoe_zone == "" || tZoe_zone == "NULL" {
		genVar.Postin <- Requestin{Node: "items", Item: "schalter_zoe_zone", Data: "t10"}
	}
	tLaden48_zone := getItemState("schalter_laden48_zone")
	if tLaden48_zone == "" || tLaden48_zone == "NULL" {
		genVar.Postin <- Requestin{Node: "items", Item: "schalter_laden48_zone", Data: "t5"}
	}
	tWaschmaschine_zone := getItemState("schalter_waschmaschine_zone")
	if tWaschmaschine_zone == "" || tWaschmaschine_zone == "NULL" {
		genVar.Postin <- Requestin{Node: "items", Item: "schalter_waschmaschine_zone", Data: "maxtotal"}
	}

	doBoiler := onOffByPrice("t4", fmt.Sprintf("%02d", hour))
	if doBoiler {
		genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_heisswasser_onoff", Data: "ON"}
		log.Println("Boiler on")
	} else {
		genVar.Postin <- Requestin{Node: "items", Item: "Zigbee_Steckdosen_steckdose_heisswasser_onoff", Data: "OFF"}
		log.Println("Boiler off")
	}

	if hour == 23 {
		os.Remove("/tmp/tibberN.json")
		os.Remove("/tmp/tibberT.json")
	}

	rules_active = true

	genVar.Telegram <- "goOpenhab initialized"
	return 0
}

// special funtions as a support to make relatively short rules

func calculateBatteryPrice(hour string) {
	var flSoc float64
	var flZone float64
	var prices []float64
	var price string
	var flPrice float64
	var hours int

	//	boolWeather := judgeWeather(4)
	boolWeather := judgePvForecast("2500")
	//soc := getItemState("Solarakku_SOC")
	soc := getSOCstr()
	zone := getItemState("Tibber_avg7")
	if zone == "0" {
		zone = getItemState("Tibber_m1")
	}
	flZone, err := strconv.ParseFloat(zone, 64)
	if err != nil {
		flZone = float64(0.25)
	}
	if flZone > float64(0.30) {
		flZone = float64(0.30)
	}
	flSoc, _ = strconv.ParseFloat(soc, 64)
	// flSoc -= 50
	flSoc -= 28
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
	genVar.Pers.Set("!BAT_PRICE", price, cache.NoExpiration)
	genVar.Postin <- Requestin{Node: "items", Item: "battery_price", Data: price}
}

func battery(cmd string) {
	switch cmd {
	case "off":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.NoExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.NoExpiration)
	case "load":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.NoExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Laden_48 switched on")
	case "loadfull":
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.NoExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Laden_48 switched on full")
	case "unload":
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.NoExpiration)
		// Soyosource on
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"ON\"}"}
		log.Println("Soyosource switched on")
	default:
		// Soyosource off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138af90599d6a/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Soyosource switched off")
		genVar.Pers.Set("Soyosource_Power_Value", "0", cache.NoExpiration)
		genVar.Postin <- Requestin{Node: "items", Item: "Soyosource_Power_Value", Data: "0"}
		// Loader-48 off
		genVar.Mqttmsg <- Mqttparms{Topic: "zigbee2mqtt/0xa4c138162567a379/set", Message: "{\"state\":\"OFF\"}"}
		log.Println("Laden_48 switched off")
		genVar.Postin <- Requestin{Node: "items", Item: "Digipot_Poti", Data: "0"}
		genVar.Pers.Set("Digipot_Poti", "0", cache.NoExpiration)
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

func judgePvForecast(search string) bool {
	var result bool = false
	x, found := genVar.Pers.Get("pv_forecast_0")
	if found {
		if x.(string) < search {
			result = true
		}
	}
	return result
}

func setCurrentPrice(h string) {
	/*
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
	   	} else {

	   		genVar.Telegram <- "goOpenhab current price not set"
	   	}
	*/
}

func getSOC() float64 {
	SOC := float64(0)
	x, found := genVar.Pers.Get("SOC")
	if found {
		soc, err := strconv.ParseFloat(x.(string), 64)
		if err == nil {
			SOC = soc
		}
	} else {
		socstr := getItemState("battery_can_SOC")
		soc, err := strconv.ParseFloat(socstr, 64)
		if err == nil {
			SOC = soc
		}
	}
	return SOC
}

func getSOCstr() string {
	SOC := string("00")
	x, found := genVar.Pers.Get("SOC")
	if found {
		SOC = x.(string)
	} else {
		SOC = getItemState("battery_can_SOC")
		genVar.Pers.Set("SOC", SOC, cache.NoExpiration)
	}
	return SOC
}

func itemToggle(item string) {
	var command string
	genVar.Getin <- Requestin{Node: "items", Item: item, Value: "state"}
	answer := <-genVar.Getout
	if answer == "ON" {
		command = "OFF"
	} else {
		command = "ON"
	}
	genVar.Postin <- Requestin{Node: "items", Item: item, Data: command}
}

func setHeating(actor string, desired string, sensor string) {
	genVar.Getin <- Requestin{Node: "items", Item: desired, Value: "state"}
	answer := <-genVar.Getout
	flDesired, _ := strconv.ParseFloat(answer, 64)
	genVar.Getin <- Requestin{Node: "items", Item: sensor, Value: "state"}
	answer = <-genVar.Getout
	flState, _ := strconv.ParseFloat(answer, 64)

	if timeVar.hour >= 7 && timeVar.hour < 12 && timeVar.weekday != time.Saturday && timeVar.weekday != time.Sunday {
		flDesired = flDesired - float64(1.2)*(float64(12*60)-float64(timeVar.dayminute))/float64(5*60)
	}
	if timeVar.hour >= 0 && timeVar.hour < 6 {
		flDesired = flDesired - float64(1.5)*(float64(6*60)-float64(timeVar.dayminute))/float64(6*60)
	}
	if timeVar.hour >= 22 && timeVar.hour < 24 {
		flDesired = flDesired - float64(1.5)
	}

	log.Printf("Heating %s: State %0.1f, Desired %0.1f\n", actor, flState, flDesired)

	if flState < flDesired {
		genVar.Postin <- Requestin{Node: "items", Item: actor, Data: "ON"}
	} else {
		genVar.Postin <- Requestin{Node: "items", Item: actor, Data: "OFF"}
	}
}
