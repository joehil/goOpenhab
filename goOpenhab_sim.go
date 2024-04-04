package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

func simMsg() {
	//	var mInfo Msginfo

	// Open the CSV file
	file, err := os.Open("yourfile.csv")
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
			if err != os.EOF { // Check if the end of file is reached
				fmt.Println("Error reading CSV data:", err)
			}
			break
		}

		// Process each field in the record
		for i, field := range record {
			fmt.Printf("Field %d: %s\n", i, field)
		}
	}
}

func simMsgs() {
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

		msgLog(mInfo)
		go processRulesInfo(mInfo)
		debugLog(5, fmt.Sprintf("Watchdog counter: %d", counter))
		mInfo.Msgevent = "watchdog.event"
		mInfo.Msgobject = "Watchdog"
		go processRulesInfo(mInfo)

	}
}
