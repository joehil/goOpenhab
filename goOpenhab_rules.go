package main

type User struct {
	id   string
	ip   string
	port uint
}

// var users []User
var users map[string]User

func processRulesInfo(mInfo *msgInfo) {
	if (mInfo.msgObject == "astro:sun:local:set#event") &&
		(mInfo.msgNewstate == "START") {
		genVar.telegram <- "Sonnenuntergang"
	}
	if (mInfo.msgObject == "astro:sun:local:rise#event") &&
		(mInfo.msgNewstate == "END") {
		genVar.telegram <- "Sonnenaufgang"
	}
}
