package main

import (
	"log"
	"time"
)

type dState struct {
	dState string
	oState string
}

var objs map[string]dState
var dNow time.Time

func putState(rin chan Requestin) {
}

func desiredState() {
	objs = make(map[string]dState)

	initObjects()

	for {
		dNow = time.Now()
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

		time.Sleep((273 * time.Second))
	}
}

func initObjects() {
	objs["Schalter_Waermepumpe_EinAus"] = dState{"ON", ""}
}

func getdState(key string) string {
	var ds string

	switch key {
	case "Schalter_Waermepumpe_EinAus":
		h := dNow.Hour()
		d := dNow.Weekday()
		ds = "ON"
		if h >= 22 || h < 5 {
			ds = "OFF"
		}
		if d == time.Weekday(6) || d == time.Weekday(0) {
			if h >= 22 || h < 6 {
				ds = "OFF"
			}
		}
	case "x":
	default:
	}

	return ds
}
