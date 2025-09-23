package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

func checkDocker() {
	time.Sleep(150 * time.Second)

	for {
		out, err := exec.Command("usr/bin/sudo", "/usr/bin/docker", "ps", "-f", "name=openhab.service", "--format", "{{.Status}}").Output()

		if err != nil {
			log.Printf("cmd.Run() failed with %s\n", err)
		}

		outStr := string(out)
		log.Printf("Docker container status: %s", outStr)
		if !strings.Contains(outStr, "healthy") && !strings.Contains(outStr, "starting") {
			log.Printf("Docker container status: %s\n", outStr)
			log.Println("Removing docker container...")
			exec.Command("usr/bin/sudo", "/usr/bin/docker", "rm", "-f", "openhab.service").Run()
			time.Sleep(60 * time.Second)
			reboot()
			log.Println("Restarting mashine...")
		}

		time.Sleep(sleepDuration + 10*time.Second)
	}
}
