package main

import (
	"fmt"
	"github.com/illarion/gonotify"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"hash/fnv"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// var read_log1 string = "/var/log/monit.log"
// var read_log2 string = "/var/log/virtualmin/remote-browser.eu_error_log"
var do_trace bool = false
var msg_trace bool = false
var pidfile string
var ownlog string
var logs []string
var rlogs []*os.File
var rpos []int64
var loghash []uint32

func main() {
	// Set location of config
	viper.SetConfigName("goOpenhab") // name of config file (without extension)
	viper.AddConfigPath("/etc/")     // path to look for the config file in

	// Read config
	read_config()

	// Get commandline args
	if len(os.Args) > 1 {
		a1 := os.Args[1]
		if a1 == "reload" {
			b, err := ioutil.ReadFile(pidfile)
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
			b, err := ioutil.ReadFile(pidfile)
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
			b, err := ioutil.ReadFile(pidfile)
			if err != nil {
				log.Fatal(err)
			}
			s := string(b)
			fmt.Println("MsgTraceOff")
			cmd := exec.Command("kill", "-12", s)
			_ = cmd.Start()
			os.Exit(0)
		}
		if a1 == "run" {
			proc_run()
		}
		fmt.Println("parameter invalid")
		os.Exit(-1)
	}
	if len(os.Args) == 1 {
		myUsage()
	}
}

func proc_run() {
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
	proc_init()

	// Catch signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
	go catch_signals(signals)

	// Open logs to read
	if do_trace {
		log.Println(logs)
	}
	for i, rlog := range logs {
		r, err := os.Open(rlog)
		if err != nil {
			log.Fatalf("error opening %s: %v", rlog, err)
		}
		rlogs = append(rlogs, r)
		defer rlogs[i].Close()
		n, err := rlogs[i].Seek(0, 2)
		rpos = append(rpos, n)
		hash := fnv.New32()
		hash.Write([]byte(rlog))
		loghash = append(loghash, hash.Sum32())
	}

	// Setup inotify watcher
	//        watcher, err := gonotify.NewFileWatcherSlice(gonotify.IN_MODIFY | gonotify.IN_MOVED_FROM | gonotify.IN_MOVED_TO | gonotify.IN_CREATE,
	//                        logs)
	watcher, err := gonotify.NewFileWatcher(gonotify.IN_MODIFY|gonotify.IN_MOVED_FROM|gonotify.IN_MOVED_TO|gonotify.IN_CREATE,
		logs[0], logs[1])

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Read watcher events
	for {
		e := <-watcher.C
		if do_trace {
			log.Println("event: ", e.Wd, e.Name, e.Mask)
		}
		switch e.Mask {
		case 2:
			rpos[e.Wd-1] = proc_log(rlogs[e.Wd-1], rpos[e.Wd-1], e.Wd-1)
		case 64:
			rlogs[e.Wd-1].Close()
		case 256:
			rlogs[e.Wd-1], err = os.Open(logs[e.Wd-1])
			if err != nil {
				log.Fatalf("error opening %s: %v", logs[e.Wd-1], err)
			}
			rpos[e.Wd-1], err = rlogs[e.Wd-1].Seek(0, 0)
		default:
			log.Println("Invalid event number")
		}
	}
}

func proc_log(f *os.File, p int64, fnr uint32) int64 {
	b1 := make([]byte, 500)
	_, _ = f.Seek(p, 0)
	m, err := f.Read(b1)
	if m > 0 && err == nil {
		t := string(b1)[:m]
		mes := strings.Split(t, "\n")
		for _, ames := range mes[:len(mes)-1] {
			// Perform customized processing due to the arrival of messages
			go process_rules(ames, fnr)
			//============================================================
			//        		if res {
			//				log.Print(ames)
			//			}
		}
		p = p + int64(m)
	}
	return p
}

// Write a pid file, but first make sure it doesn't exist with a running pid.
func writePidFile(pidFile string) error {
	// Read in the pid file as a slice of bytes.
	if piddata, err := ioutil.ReadFile(pidFile); err == nil {
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
	return ioutil.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
}

func catch_signals(c <-chan os.Signal) {
	for {
		s := <-c
		log.Println("Got signal:", s)
		if s == syscall.SIGHUP {
			read_config()
			read_users()
		}
		if s == syscall.SIGUSR1 {
			msg_trace = true
			log.Println("msg_trace switched on")
		}
		if s == syscall.SIGUSR2 {
			msg_trace = false
			log.Println("msg_trace switched off")
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

	if do_trace {
		log.Println("do_trace: ", do_trace)
		log.Println("own_log; ", ownlog)
		log.Println("pid_file: ", pidfile)
		for i, v := range logs {
			log.Printf("Index: %d, Value: %v\n", i, v)
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
}
