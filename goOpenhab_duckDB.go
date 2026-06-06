package main

import (
	"database/sql"
	"log"

	_ "github.com/duckdb/duckdb-go/v2"
)

func duckDB_init() {
	db, err := sql.Open("duckdb", duckDBPath)
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS energy
		(tstamp TIMESTAMP_MS,
		device VARCHAR NOT NULL,
		packInputPower BIGINT,
		outputPackPower BIGINT,
		outputHomePower BIGINT,
		solarInputPower BIGINT,
		soc BIGINT, PRIMARY KEY("tstamp"))`)
	if err != nil {
		log.Println(err)
	}
}

func duckDB_insert(device string, packInputPower int64, outputPackPower int64,
	outputHomePower int64, solarInputPower int64, soc int64) {
	db, err := sql.Open("duckdb", duckDBPath)
	if err != nil {
		log.Println(err)
	}
	defer db.Close()
	/*
	   _, err = db.Exec(`INSERT INTO main.energy

	   	(tstamp, device, packInputPower, outputPackPower, outputHomePower, solarInputPower, soc)
	   	values (NOW(), device, packInputPower,
	   	outputPackPower, outputHomePower, solarInputPower, soc)`)
	   	log.Println(err)
	*/
}
