package main

import (
	"log"
	"os/exec"
)

func suppress_field(nr int, word string, do_log bool, fields []string) bool {
	if len(fields) > nr {
		if fields[nr] == word {
			return false
		}
	}
	return do_log
}

func exec_cmd(command string, args ...string) {
	cmd := exec.Command(command, args...)
	err := cmd.Run()
	if err != nil {
		log.Printf("Command finished with error: %v", err)
	}
}

func getItemState(item string) string {
	var answer string = ""
	if x, found := genVar.Pers.Get(item); found {
                answer = x.(string)
        } else {
                genVar.Getin <- Requestin{Node: "items", Item: item, Value: "state"}
                answer = <-genVar.Getout
                if answer != "" {
                        genVar.Putin <- Requestin{Node: "items", Item: item, Value: "state", Data: answer}
                }
	}
	return answer
}
