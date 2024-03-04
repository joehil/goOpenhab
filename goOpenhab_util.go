package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
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

func traceLog(message string) {
	if do_trace {
		log.Println(message)
	}
}

func debugLog(sev int, message string) {
	if sev >= logseverity {
		log.Println(message)
	}
}

func msgLog(minfo Msginfo) {
	if msg_trace {
		fmt.Fprintf(dfile, "%s;%s;%s;%s;%s\n", minfo.Msgevent, minfo.Msgobjtype, minfo.Msgobject, minfo.Msgoldstate, minfo.Msgnewstate)
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
			genVar.Postin <- Requestin{Node: "items", Item: item, Value: "state", Data: answer}
		}
	}
	return answer
}

func wlanTraffic() int {
	var answer int = 0
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/ifconfig", "-s", "wlan0")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("error processing command ifconfig:", err)
		return -99
	}
	//	fmt.Println("output of ifconfig:\n", out.String())
	parts := strings.Split(out.String(), "\n")
	if len(parts) > 0 {
		//		fmt.Println(parts[1])
		words := strings.Fields(parts[1])
		//		fmt.Println("words ", len(words), words)
		if len(words) > 5 {
			rxNew := words[2]
			txNew := words[6]
			rxOld := getItemState("!WLANRX")
			txOld := getItemState("!WLANTX")
			fmt.Println("network values :", rxOld, txOld, rxNew, txNew)
			if rxOld != rxNew || txOld != txNew {
				answer = 1
			}
			genVar.Pers.Set("!WLANRX", rxNew, cache.DefaultExpiration)
			genVar.Pers.Set("!WLANTX", txNew, cache.DefaultExpiration)
		}
	}
	return answer
}

func restartNetwork() {
	cmd1 := exec.Command("/usr/bin/sudo", "/usr/sbin/service", "networking", "stop")
	cmd1.Run()
	time.Sleep(10 * time.Second)
	cmd2 := exec.Command("/usr/bin/sudo", "/usr/sbin/service", "networking", "start")
	cmd2.Run()
}

func reboot() {
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/reboot")
	cmd.Run()
}
