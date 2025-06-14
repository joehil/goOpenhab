package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
	if logseverity >= sev {
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
		debugLog(6, "item state: "+answer)
	} else {
		genVar.Getin <- Requestin{Node: "items", Item: item, Value: "state"}
		answer = <-genVar.Getout
		//if answer != "" {
		//	genVar.Pers.Set(item, answer, cache.DefaultExpiration)
		//}
	}
	return answer
}

func restartNetwork() {
	cmd1 := exec.Command("/usr/bin/sudo", "/usr/bin/nmcli", "n", "off")
	cmd1.Run()
	time.Sleep(10 * time.Second)
	cmd2 := exec.Command("/usr/bin/sudo", "/usr/bin/nmcli", "n", "on")
	cmd2.Run()
}

func reboot() {
	cmd := exec.Command("/usr/bin/sudo", "/usr/sbin/reboot")
	cmd.Run()
}

func createMessage(event string, object string, status string) {
	var mInfo Msginfo

	hours, minutes, seconds := time.Now().Clock()

	currentTime := time.Now()
	tdat := fmt.Sprintf("%04d-%02d-%02d",
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day())

	mInfo.Msgdate = tdat
	mInfo.Msgtime = fmt.Sprintf("%02d:%02d:%02d.000", hours, minutes, seconds)
	mInfo.Msgevent = event
	mInfo.Msgobject = object
	mInfo.Msgnewstate = status

	go processRulesInfo(mInfo)
}

func readJson(inJson string, field string) string {
	jsonData := []byte(inJson)

	var result map[string]interface{}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		log.Printf("error unmarshalling JSON: %v", err)
	}

	if outJson, ok := result[field].(string); ok {
		return outJson
	} else {
		return ""
	}
}

func dimmerBrightness(device string, change int) string {
	var brightness int = 127
	if x, found := genVar.Pers.Get("!" + device); found {
		brightness = x.(int)
	}
	brightness += change
	if brightness < 0 {
		brightness = 0
	}
	if brightness > 250 {
		brightness = 250
	}
	genVar.Pers.Set("!"+device, brightness, cache.NoExpiration)
	return fmt.Sprintf("%d", brightness)
}

func dimmerKnob(mInfo Msginfo, deviceName string, deviceAction string, dimmerDevice string, toDimDevice string) {
	if mInfo.Msgobject == dimmerDevice {
		debugLog(6, "Dimmer device:"+dimmerDevice+" newstate: "+mInfo.Msgnewstate+" action: "+deviceAction)
		switch readJson(mInfo.Msgnewstate, deviceAction) {
		case "single":
			// switch light on half brightness
			genVar.Mqttmsg <- Mqttparms{Topic: toDimDevice, Message: "{\"state\":\"ON\",\"brightness\":80}"}
			debugLog(5, deviceName+" on, cozy")
		case "double":
			// switch light off
			genVar.Mqttmsg <- Mqttparms{Topic: toDimDevice, Message: "{\"state\":\"OFF\"}"}
			debugLog(5, deviceName+" off")
		case "hold":
			// switch light on full brightness
			genVar.Mqttmsg <- Mqttparms{Topic: toDimDevice, Message: "{\"state\":\"ON\",\"brightness\":250}"}
			debugLog(5, deviceName+" on, full")
		case "rotate_right":
			// make light brighter
			br := dimmerBrightness(deviceName, 10)
			genVar.Mqttmsg <- Mqttparms{Topic: toDimDevice, Message: "{\"brightness\":\"" + br + "\"}"}
			debugLog(5, deviceName+" brighter")
		case "rotate_left":
			// make light less bright
			br := dimmerBrightness(deviceName, -10)
			genVar.Mqttmsg <- Mqttparms{Topic: toDimDevice, Message: "{\"brightness\":\"" + br + "\"}"}
			debugLog(5, deviceName+" less bright")
		default:
		}
		return
	}
}

func setItemAlarmTime(item string, alarmtime int) {
	var name string = "!ALARM_" + item
	var recovery string = "!RECOVERY_" + item
	d := time.Now().Unix() + int64(alarmtime)
	//	genVar.Pers.Delete(name)
	genVar.Pers.Set(name, fmt.Sprintf("%d", d), cache.NoExpiration)
	if _, found := genVar.Pers.Get(recovery); found {
		genVar.Telegram <- recovery
		genVar.Pers.Delete(recovery)
	}
}

func checkItemAlarm(item string) bool {
	var name string = "!ALARM_" + item
	var answer bool = false
	d := time.Now().Unix()
	var tim string
	var alarm int64
	if x, found := genVar.Pers.Get(name); found {
		tim = x.(string)
		al, err := strconv.ParseInt(tim, 10, 64)
		if err != nil {
			alarm = d
		} else {
			alarm = al
		}
	}

	if d > alarm {
		answer = true
	}
	return answer
}

func iterateAlarms() {
	debugLog(7, "iterateAlarms")
	alarms := genVar.Pers.Items()
	for k, v := range alarms {
		if len(k) >= 7 {
			if k[0:7] == "!ALARM_" {
				item := k[7:]
				if checkItemAlarm(item) {
					log.Printf("key[%s] value[%v]\n", item, v)
					genVar.Telegram <- k
					genVar.Pers.Delete(k)
					genVar.Pers.Set("!RECOVERY_"+item, "0", cache.NoExpiration)
				}
			}
		}
	}
}

func setItemOffTime(item string, offtime int) {
	var name string = "!OFF_" + item
	d := time.Now().Unix() + int64(offtime)
	if offtime > 0 {
		genVar.Pers.Set(name, fmt.Sprintf("%d", d), cache.NoExpiration)
	} else {
		genVar.Pers.Delete(name)
	}
}

func checkItemOff(item string) bool {
	var name string = "!OFF_" + item
	var answer bool = false
	d := time.Now().Unix()
	var tim string
	var off int64
	if x, found := genVar.Pers.Get(name); found {
		tim = x.(string)
		al, err := strconv.ParseInt(tim, 10, 64)
		if err != nil {
			off = d
		} else {
			off = al
		}
	}

	if d > off {
		answer = true
	}
	return answer
}

func iterateOffs() {
	debugLog(7, "iterateOffs")
	offs := genVar.Pers.Items()
	for k, v := range offs {
		if len(k) >= 5 {
			if k[0:5] == "!OFF_" {
				item := k[5:]
				if checkItemOff(item) {
					log.Printf("key[%s] value[%v]\n", item, v)
					genVar.Postin <- Requestin{Node: "items", Item: item, Data: "OFF"}
					genVar.Pers.Delete(k)
				}
			}
		}
	}
}

func setFHEM(device string, value string) {
	requestURL := fmt.Sprintf("http://192.168.0.211:8083/fhem?cmd=set%%20%s%%20%s", device, value)
	log.Println(requestURL)
	_, err := http.Get(requestURL)
	//	_, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Printf("client: could not create request: %s\n", err)
	}
}

// getSystemUptime returns the system uptime in seconds (Linux only)
func getSystemUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected content in /proc/uptime")
	}
	return strconv.ParseFloat(fields[0], 64)
}
