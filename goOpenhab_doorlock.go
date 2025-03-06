package main

import (
	"encoding/hex"
	"log"
	"math/rand"
	"strings"
	"time"
)

var rNum [10]byte
var rNumber int

func connect2Doorlock(secrets []int, tags []string, pwds []string) {
	for {
		creKey()
		genVar.Mqttmsg <- Mqttparms{Topic: "doorlock/in/cls", Message: "X"}
		time.Sleep(time.Second)
		for _, element := range tags {
			strs := strings.Split(element, ";")
			if len(strs[0]) == 29 {
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
		for _, element := range pwds {
			crypted := xcrypt([]byte(element), secrets)
			strcrypted := string(crypted)
			strrnum := string(rNum[:])
			genVar.Mqttmsg <- Mqttparms{Topic: "doorlock/in/pwd/add", Message: strrnum + strcrypted}
			time.Sleep(time.Second)
		}

		time.Sleep(15 * time.Minute)
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
