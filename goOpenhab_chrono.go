package main

import (
	"fmt"
	"time"

	"github.com/joehil/jhtype"
)

func timeTrigger() {
	for {
		var mInfo jhtype.Msginfo
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
		if usePlugin {
			go tryRulesFunc(mInfo, ptrGenVars)
		}
	}
}
