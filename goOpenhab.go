package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/nxadm/tail"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
)

type msgInfo struct {
	msgDate     string
	msgTime     string
	msgEvent    string
	msgObjtype  string
	msgObject   string
	msgOldstat  string
	msgNewstate string
}

type msgWarn struct {
	msgDate  string
	msgTime  string
	msgEvent string
	msgText  string
}

type generalVars struct {
	pers     *cache.Cache
	telegram chan string
	tbtoken  string
	chatid   int64
}

var do_trace bool = false
var msg_trace bool = false
var pidfile string
var ownlog string
var logs []string
var rlogs []*os.File
var rpos []int64
var loghash []uint32
var timeOld time.Time

var genVar *generalVars = new(generalVars)

func main() {
	// Set location of config
	viper.SetConfigName("goOpenhab") // name of config file (without extension)
	viper.AddConfigPath("/etc/")     // path to look for the config file in

	// Read config
	read_config()

	timeOld = time.Now()

	genVar.pers = cache.New(3*time.Hour, 10*time.Hour)
	traceLog("Persistence was initialized")

	genVar.telegram = make(chan string)

	go sendTelegram(genVar.telegram)
	traceLog("Telegram interface was initialized")

	// Get commandline args
	if len(os.Args) > 1 {
		a1 := os.Args[1]
		if a1 == "reload" {
			b, err := os.ReadFile(pidfile)
			if err != nil {
				log.Fatal(err)
			}
			s := string(b)
			fmt.Println("Reload", s)
			cmd := exec.Command("kill", "-hup", s)
			_ = cmd.Start()
			os.Exit(0)
		}
		if a1 == "mtraceon" {
			b, err := os.ReadFile(pidfile)
			if err != nil {
				log.Fatal(err)
			}
			s := string(b)
			fmt.Println("MsgTraceOn")
			cmd := exec.Command("kill", "-10", s)
			_ = cmd.Start()
			os.Exit(0)
		}
		if a1 == "mtraceoff" {
			b, err := os.ReadFile(pidfile)
			if err != nil {
				log.Fatal(err)
			}
			s := string(b)
			fmt.Println("MsgTraceOff")
			cmd := exec.Command("kill", "-12", s)
			_ = cmd.Start()
			os.Exit(0)
		}
		if a1 == "stop" {
			b, err := os.ReadFile(pidfile)
			if err != nil {
				log.Fatal(err)
			}
			s := string(b)
			fmt.Println("Stop goOpenhab")
			cmd := exec.Command("kill", "-9", s)
			_ = cmd.Start()
			os.Exit(0)
		}
		if a1 == "run" {
			procRun()
		}
		fmt.Println("parameter invalid")
		os.Exit(-1)
	}
	if len(os.Args) == 1 {
		myUsage()
	}
}

func procRun() {
	// Write pidfile
	err := writePidFile(pidfile)
	if err != nil {
		log.Fatalf("Pidfile could not be written: %v", err)
	}
	defer os.Remove(pidfile)

	// Open log file
	ownlogger := &lumberjack.Logger{
		Filename:   ownlog,
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}
	defer ownlogger.Close()
	log.SetOutput(ownlogger)

	// Inform about trace
	log.Println("Trace set to: ", do_trace)

	// Do customized initialization
	//proc_init()

	// Catch signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGKILL)
	go catch_signals(signals)

	// Open logs to read
	if do_trace {
		log.Println(logs)
	}
	for _, rlog := range logs {
		traceLog("Task started for " + rlog)
		go tailLog(rlog)
	}
	for {
		time.Sleep(10 * time.Second)
	}
}

func traceLog(message string) {
	if do_trace {
		log.Println(message)
	}
}

func msgLog(message string) {
	if msg_trace {
		log.Println(message)
	}
}

func procLine(msg string) {
	var mInfo *msgInfo = new(msgInfo)
	var mWarn *msgWarn = new(msgWarn)
	if len(msg) > 75 {
		msgType := msg[25:29]
		if msgType == "INFO" {
			mInfo.msgDate = msg[0:10]
			mInfo.msgTime = msg[11:23]
			mInfo.msgEvent = msg[33:69]
			rest := msg[73:]
			mes := strings.Split(rest, " ")
			if mInfo.msgEvent == "openhab.event.ItemStateChangedEvent" {
				if len(mes) == 7 {
					mInfo.msgObjtype = mes[0]
					mInfo.msgObject = mes[1]
					mInfo.msgOldstat = mes[4]
					mInfo.msgNewstate = mes[6]
				}
				if len(mes) == 9 {
					mInfo.msgObjtype = mes[0]
					mInfo.msgObject = mes[1]
					mInfo.msgOldstat = strings.Join(mes[5:5], " ")
					mInfo.msgNewstate = strings.Join(mes[7:8], " ")
				}
				//			fmt.Println("012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789")
				//			fmt.Println(msg)
				//fmt.Println(mInfo)
			}
			if mInfo.msgEvent == "openhab.event.ChannelTriggeredEvent" {
				if len(mes) >= 3 {
					mInfo.msgObject = mes[0]
					mInfo.msgNewstate = mes[2]
				}
				//			fmt.Println("012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789")
				//			fmt.Println(msg)
				//fmt.Println(mInfo)
			}

			processRulesInfo(mInfo)
		}
		if msgType == "WARN" {
			mWarn.msgDate = msg[0:10]
			mWarn.msgTime = msg[11:23]
			mWarn.msgEvent = msg[33:69]
			mWarn.msgText = msg[73:]
			//			fmt.Println("012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789")
			//			fmt.Println(msg)
			//fmt.Println(mWarn)
		}
	}
}

// Write a pid file, but first make sure it doesn't exist with a running pid.
func writePidFile(pidFile string) error {
	// Read in the pid file as a slice of bytes.
	if piddata, err := os.ReadFile(pidFile); err == nil {
		// Convert the file contents to an integer.
		if pid, err := strconv.Atoi(string(piddata)); err == nil {
			// Look for the pid in the process list.
			if process, err := os.FindProcess(pid); err == nil {
				// Send the process a signal zero kill.
				if err := process.Signal(syscall.Signal(0)); err == nil {
					// We only get an error if the pid isn't running, or it's not ours.
					return fmt.Errorf("pid already running: %d", pid)
				}
			}
		}
	}
	// If we get here, then the pidfile didn't exist,
	// or the pid in it doesn't belong to the user running this app.
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
}

func catch_signals(c <-chan os.Signal) {
	for {
		s := <-c
		log.Println("Got signal:", s)
		if s == syscall.SIGHUP {
			read_config()
			//			read_users()
		}
		if s == syscall.SIGUSR1 {
			msg_trace = true
			log.Println("msg_trace switched on")
		}
		if s == syscall.SIGUSR2 {
			msg_trace = false
			log.Println("msg_trace switched off")
		}
		if s == syscall.SIGKILL {
			log.Println("msg_trace switched off")
			os.Exit(0)
		}
	}
}

func read_config() {
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Config file not found: %v", err)
	}

	pidfile = viper.GetString("pid_file")
	if pidfile == "" { // Handle errors reading the config file
		log.Fatalf("Filename for pidfile unknown: %v", err)
	}
	ownlog = viper.GetString("own_log")
	if ownlog == "" { // Handle errors reading the config file
		log.Fatalf("Filename for ownlog unknown: %v", err)
	}
	logs = viper.GetStringSlice("logs")
	do_trace = viper.GetBool("do_trace")
	genVar.tbtoken = viper.GetString("tbtoken")
	genVar.chatid = int64(viper.GetInt("chatid"))

	if do_trace {
		log.Println("do_trace: ", do_trace)
		log.Println("own_log; ", ownlog)
		log.Println("pid_file: ", pidfile)
		for i, v := range logs {
			log.Printf("Index: %d, Value: %v\n", i, v)
		}
	}
}

func tailLog(logFile string) {
	t, err := tail.TailFile(logFile, tail.Config{Follow: true})
	if err != nil {
		panic(err)
	}
	for line := range t.Lines {
		tNow := time.Now()
		if tNow.Sub(timeOld) > time.Second {
			msgLog(line.Text)
			go procLine(line.Text)
		}
	}
}

func myUsage() {
	fmt.Printf("Usage: %s argument\n", os.Args[0])
	fmt.Println("Arguments:")
	fmt.Println("run           Run progam as daemon")
	fmt.Println("reload        Make running daemon reload it's configuration")
	fmt.Println("mtraceon      Make running daemon switch it's message tracing on (useful for coding new rules)")
	fmt.Println("mtraceoff     Make running daemon switch it's message tracing off")
	fmt.Println("stop          Stop daemon")
}
