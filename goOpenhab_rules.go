package main

import (
	"fmt"
	"strings"

	"github.com/patrickmn/go-cache"
)

func processRulesInfo(mInfo Msginfo) {
	if mInfo.Msgevent == "chrono.event" {
		chronoEvents(mInfo)
		return
	}

	if len(mInfo.Msgobject) >= 8 {
		if mInfo.Msgobject[0:8] == "Tibber_t" {
			fmt.Println(mInfo.Msgobject, mInfo.Msgnewstate)
			genVar.Pers.Set(mInfo.Msgobject, mInfo.Msgnewstate, cache.DefaultExpiration)
		}
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
		return
	}
}
