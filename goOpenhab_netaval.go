package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const networkUnavailableCode = 999

// isNetworkAvailable checks the availability of a network resource by sending
// a GET request to the specified URL. It uses a context with a timeout of 5 seconds
// to ensure the request does not hang indefinitely. Returns an HTTP status code
// indicating the result of the network check.
func isNetworkAvailable(url string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return networkUnavailableCode
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return networkUnavailableCode
	}
	defer resp.Body.Close()

	return http.StatusOK
}

/*
checkNetwork continuously monitors the availability of machine, local, and internet networks.
It checks each network's status every 10 seconds and logs any availability errors.
If a network is unavailable for more than three consecutive checks, it triggers a message event.
When all networks are available, it logs a success message and sends an "all ok" event.
The function runs indefinitely, pausing for 2 minutes between each complete cycle.
*/
func checkNetwork() {
	var avalLocal int
	var avalMachine int
	var avalInternet int
	var cntMachine int = 0
	var cntLocal int = 0
	var cntInternet int = 0
	for {
		avalMachine = isNetworkAvailable(genVar.machineNet)
		time.Sleep(10 * time.Second)
		avalLocal = isNetworkAvailable(genVar.localNet)
		time.Sleep(10 * time.Second)
		avalInternet = isNetworkAvailable(genVar.interNet)

		switch {
		case avalMachine != http.StatusOK:
			traceLog(fmt.Sprintf("Network availability error: %s %d", genVar.machineNet, avalMachine))
			cntMachine++
			if cntMachine > 3 {
				createMessage("network.availability.machine.event", genVar.machineNet, fmt.Sprintf("%d", avalMachine))
			}
		default:
			cntMachine = 0
		}

		switch {
		case avalLocal != http.StatusOK:
			traceLog(fmt.Sprintf("Network availability error: %s %d", genVar.localNet, avalLocal))
			cntLocal++
			if cntLocal > 3 {
				createMessage("network.availability.local.event", genVar.localNet, fmt.Sprintf("%d", avalLocal))
			}
		default:
			cntLocal = 0
		}

		switch {
		case avalInternet != http.StatusOK:
			traceLog(fmt.Sprintf("Network availability error: %s %d", genVar.interNet, avalInternet))
			cntInternet++
			if cntInternet > 3 {
				createMessage("network.availability.internet.event", genVar.interNet, fmt.Sprintf("%d", avalInternet))
			}
		default:
			cntInternet = 0
		}

		if avalMachine == http.StatusOK && avalLocal == http.StatusOK && avalInternet == http.StatusOK {
			traceLog("Network availability ok")
			createMessage("network.availability.all.event", "all", "ok")
		}
		time.Sleep(2 * time.Minute)
	}
}
