package main

import (
	"strings"
	"os"
	"io"
	"encoding/csv"
	"bufio"
	"log"
	"strconv"
)

type User struct {
    id string
    ip string
    port uint
}

//var users []User
var users map[string]User


// Do customized initialisation
func proc_init() {
	read_users()
}

func process_rules(message string, lognr uint32) {
	var do_log bool = true
	fields := strings.Fields(message)
	logname := logs[lognr]
	lhash := loghash[lognr]

	if msg_trace {
		log.Println("lognr:", lognr, "lhash:", lhash, "logname:", logname)
		for i, field := range fields {
			log.Println(i,": ",field)
		}
	}

	if len(fields) > 16 {
		if fields[10] == "AH00957:" {
			f := fields[16]
			s := strings.Split(f, ":")
			f = s[1]
			log.Println(users[f].id, "is being started")
			exec_cmd("/usr/bin/monit", "start", users[f].id)
		}
	}

        if len(fields) > 8 {
                if fields[4] == "error" && fields[8] == "cpu" {
                        f := fields[6]
                        f = strings.Replace(f, "'", "", 2)
                        log.Println(f, "is being stopped")
                        exec_cmd("/usr/bin/monit", "stop", f)
                }
        }

        if len(fields) > 12 {
                if fields[4] == "error" && fields[12] == "125)" {
                        f := fields[6]
                        f = strings.Replace(f, "'", "", 2)
                        log.Println(f, " container is being removed")
                        exec_cmd("/usr/bin/docker", "rm", f)
                }
        }

	do_log = suppress_field(10, "AH00126:", do_log, fields)

        do_log = suppress_field(10, "AH01114:", do_log, fields)

        do_log = suppress_field(10, "AH01276:", do_log, fields)

        do_log = suppress_field(10, "AH02811:", do_log, fields)

	if do_log {
		log.Println(message)
	}
}

// Function do read in a custom map to make special decisions 
func read_users() {
    csvFile, _ := os.Open("/root/ibrowser-helper/envs.dat")
    reader := csv.NewReader(bufio.NewReader(csvFile))
    reader.Comma = rune(59)
    users = make(map[string]User)
    for {
        line, error := reader.Read()
        if error == io.EOF {
            break
        } else if error != nil {
            log.Fatal(error)
        }
        i, err := strconv.Atoi(line[2])
        if err != nil {
                // handle error
                log.Fatal(err)
                os.Exit(2)
        }
        users[line[2]] = User{
            id: line[0],
            ip:  line[1],
            port: uint(i),
        }
    }
    log.Println("User data read")
    csvFile.Close()
    if do_trace {
            log.Println(users)
    }
}
