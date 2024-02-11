package main

import "fmt"

type User struct {
	id   string
	ip   string
	port uint
}

// var users []User
var users map[string]User

func processRulesInfo(mInfo *msgInfo) {
	if (mInfo.msgObject == "'XXXHeizung_unten_Temperatur'") ||
		(mInfo.msgObject == "'XXXCPU_Last'") {
		genVar.telegram <- "Test Telegram"
		fmt.Print(mInfo.msgTime)
		fmt.Println(" test telegram")
	}
}
