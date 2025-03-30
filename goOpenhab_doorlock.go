package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

var rNum [10]byte
var rNumber int
var min int
var hour string
var day string

func connect2Doorlock(secrets []int, tags *[]string, pwds *[]string) {
	for {
		min = time.Now().Minute()
		if min%15 == 0 {
			creKey()
			genVar.Mqttmsg <- Mqttparms{Topic: "doorlock/in/cls", Message: "X"}
			time.Sleep(time.Second)
			for _, element := range *tags {
				strs := strings.Split(element, ";")
				hour = fmt.Sprintf("%0d", time.Now().Hour())
				day = fmt.Sprintf("%0d", time.Now().Weekday())
				if len(strs[0]) == 29 &&
					(strs[1] == "*" || strings.Contains(strs[1], day)) &&
					(strs[2] == "*" || strings.Contains(strs[2], hour)) {
					atag := strings.ReplaceAll(strs[0], ":", "")
					decoded, err := hex.DecodeString(atag)
					if err != nil {
						log.Fatal(err)
					} else {
						crypted := xcrypt(decoded, secrets)
						strcrypted := string(crypted)
						strrnum := string(rNum[:])
						genVar.Mqttmsg <- Mqttparms{Topic: "doorlock/in/tag/add", Message: strrnum + strcrypted}
					}
				}
				time.Sleep(time.Second)
			}
			for _, element := range *pwds {
				crypted := xcrypt([]byte(element), secrets)
				strcrypted := string(crypted)
				strrnum := string(rNum[:])
				genVar.Mqttmsg <- Mqttparms{Topic: "doorlock/in/pwd/add", Message: strrnum + strcrypted}
				time.Sleep(time.Second)
			}
		}
		time.Sleep(time.Minute)
	}
}

func xcrypt(msg []byte, s []int) []byte {
	for i, _ := range msg {
		msg[i] = msg[i] ^ byte(s[rNumber+i])
	}
	return msg
}

func creKey() {
	rNumber = 0
	for i, _ := range rNum {
		rNum[i] = byte(rand.Intn(255))
		rNumber += int(rNum[i])
	}
	rNumber %= 255
}
