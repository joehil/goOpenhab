package main

import (
	"github.com/joehil/jhtype"
//	"fmt"
)

func processRulesInfo(mInfo jhtype.Msginfo) {
//	fmt.Println(genVar.Telegram)
	if len(mInfo.Msgobject) >= 5 {
		if mInfo.Msgobject[0:5] == "astro" {
			genVar.Telegram <- "Astro Event"
		}
	}
	if (mInfo.Msgobject == "astro:sun:local:rise#event") &&
		(mInfo.Msgnewstate == "END") {
		genVar.Telegram <- "Sonnenaufgang"
	}
}
