package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

func simMsg() {
	var mInfo Msginfo

	genVar.Pers = cache.New(3*time.Hour, 10*time.Hour)
	traceLog("Persistence was initialized")

	genVar.Telegram = make(chan string)

	go sendTelegram(genVar.Telegram)
	traceLog("Telegram interface was initialized")

	genVar.Mqttmsg = make(chan Mqttparms, 5)

	go publishMqtt(genVar.Mqttmsg)
	traceLog("MQTT interface was initialized")

	genVar.Getin = make(chan Requestin)
	genVar.Getout = make(chan string)

	go restApiGet(genVar.Getin, genVar.Getout)
	traceLog("restapi get interface was initialized")

	genVar.Postin = make(chan Requestin)

	go restApiPost(genVar.Postin)
	traceLog("restapi post interface was initialized")

	go timeTrigger()
	traceLog("chrono server was initialized")

	// Open the CSV file
	file, err := os.Open(dumpfile)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	defer file.Close()

	// Create a new reader from the file
	reader := csv.NewReader(file)

	// Read and process records one line at a time
	for {
		record, err := reader.Read()
		if err != nil {
			if err == csv.ErrFieldCount { // Handle expected number of fields error
				fmt.Println("Warning: unexpected number of fields in line", err)
				continue
			}
			if err != io.EOF { // Check if the end of file is reached
				fmt.Println("Error reading CSV data:", err)
			}
			break
		}

		// Process each field in the record
		for _, rec := range record {
			hours, minutes, seconds := time.Now().Clock()
			currentTime := time.Now()
			tdat := fmt.Sprintf("%04d-%02d-%02d",
				currentTime.Year(),
				currentTime.Month(),
				currentTime.Day())
			mInfo.Msgdate = tdat
			mInfo.Msgtime = fmt.Sprintf("%02d:%02d:%02d.000", hours, minutes, seconds)
			field := strings.Split(rec, ";")

			mInfo.Msgevent = field[0]
			mInfo.Msgobjtype = field[1]
			mInfo.Msgobject = field[2]
			mInfo.Msgoldstate = field[3]
			mInfo.Msgnewstate = field[4]

			fmt.Println("==>", mInfo)
			go processRulesInfo(mInfo)
			time.Sleep(time.Second)
			counter++
		}
	}
	fmt.Println("===== Simulation finished =====")
}
