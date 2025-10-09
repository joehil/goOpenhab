package main

import (
	"fmt"
	"time"
)

func timeTrigger() {
	var secs int
	var old uint64
	_, _, secs = time.Now().Clock()
	for secs != 0 {
		time.Sleep(1 * time.Second)
		_, _, secs = time.Now().Clock()
		chronoCounter++
	}
	for {
		var mInfo Msginfo
		time.Sleep(1 * time.Minute)
		hours, minutes, seconds := time.Now().Clock()

		currentTime := time.Now()
		tdat := fmt.Sprintf("%04d-%02d-%02d",
			currentTime.Year(),
			currentTime.Month(),
			currentTime.Day())

		mInfo.Msgdate = tdat
		mInfo.Msgtime = fmt.Sprintf("%02d:%02d:%02d.000", hours, minutes, seconds)
		mInfo.Msgevent = "chrono.event"
		mInfo.Msgobject = fmt.Sprintf("%02d:%02d", hours, minutes)

		go processRulesInfo(mInfo)
		debugLog(5, fmt.Sprintf("Watchdog counter: %d", counter))
		if counter == old && rules_active {
			mInfo.Msgevent = "watchdog.event"
			mInfo.Msgobject = "Watchdog Chrono"
			go processRulesInfo(mInfo)
		}

		if minutes == 0 || minutes == 15 || minutes == 30 || minutes == 45 {
			mInfo.Msgevent = "periodic15.event"
			mInfo.Msgobject = "Periodic 15min"
			go processRulesInfo(mInfo)
		}
		old = counter
		chronoCounter++
	}
}
