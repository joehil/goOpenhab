package main

import (
	"log"
	"strconv"
	"sync"
	"time"
)

type desState struct {
	dState string
	oState string
}

var (
	objs  map[string]desState
	dNow  time.Time
	mutex sync.Mutex
)

const (
	sleepDuration = 273 * time.Second
	tempThreshold = -4.0
)

func putState(obj string, oState string) {
	mutex.Lock()
	defer mutex.Unlock()

	log.Println("Updating state for:", obj)
	x := desState{
		dState: objs[obj].dState,
		oState: oState,
	}
	objs[obj] = x
	log.Printf("oState for %s set to %s\n", obj, oState)
}

func desiredState() {
	objs = make(map[string]desState)

	initObjects()

	for {
		dNow = time.Now()
		mutex.Lock()
		for key, obj := range objs {
			dStat := obj.oState
			if obj.oState == "" {
				ds := getdState(key)
				obj.dState = ds
				dStat = ds
			}
			aStat := getItemState(key)
			log.Println(key, "desired:", dStat, "actual:", aStat)
			if aStat != dStat {
				genVar.Postin <- Requestin{Node: "items", Item: key, Value: "state", Data: dStat}
				log.Println(key + " = " + dStat)
			}
		}
		mutex.Unlock()

		time.Sleep(sleepDuration)
	}
}

func initObjects() {
	objs["Schalter_Waermepumpe_EinAus"] = desState{"ON", ""}
}

func getdState(key string) string {
	var ds string

	switch key {
	case "Schalter_Waermepumpe_EinAus":
		h := dNow.Hour()
		d := dNow.Weekday()
		strTemp := getItemState("Thermometer_Warmepumpe_Temperature")
		flTemp, err := strconv.ParseFloat(strTemp, 64)
		if err != nil {
			log.Printf("Error parsing temperature: %v", err)
			flTemp = 0
		}
		ds = "ON"
		if flTemp > tempThreshold {
			if h >= 22 || h < 5 {
				ds = "OFF"
			}
			if d == time.Saturday || d == time.Sunday {
				if h >= 22 || h < 6 {
					ds = "OFF"
				}
			}
		}
	default:
		log.Printf("Unknown key: %s", key)
	}

	return ds
}
