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

/*
Package main implements a system for managing and updating the desired state of objects
based on certain conditions such as time and temperature. It includes functions to update
object states, determine desired states, and initialize objects with default states.

Types:
- desState: Represents the desired and actual state of an object.

Variables:
- objs: A map storing the desired states of objects.
- dNow: The current time.
- mutex: A mutex for synchronizing access to shared resources.

Constants:
- sleepDuration: The duration to wait between state checks.
- tempThreshold: The temperature threshold for determining the desired state.

Functions:
- putState: Updates the operational state of a specified object.
- desiredState: Continuously checks and updates the desired state of objects.
- initObjects: Initializes objects with default desired states.
- getdState: Determines the desired state of an object based on time and temperature.
*/
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

/*
desiredState continuously monitors and updates the state of objects.

This function initializes a map of objects and enters an infinite loop where it:
1. Locks a mutex to ensure thread safety.
2. Iterates over each object to compare its desired state with the actual state.
3. Updates the desired state if it is not set.
4. Logs the desired and actual states.
5. Sends a request to update the state if there is a discrepancy.
6. Unlocks the mutex and sleeps for a specified duration before repeating.

The function relies on external functions and variables such as initObjects, getdState, getItemState, and genVar.Postin.
*/
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
