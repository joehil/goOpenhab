package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func restApiGet(rin chan Requestin, rout chan string) {
	url := genVar.Resturl     // Die URL der API, die du aufrufen möchtest
	token := genVar.Resttoken // Der Bearer Token für die Authentifizierung

	for {
		request := <-rin
		requrl := url + "/" + request.Node + "/" + request.Item + "/" + request.Value
		// Erstelle einen neuen Request
		req, err := http.NewRequest("GET", requrl, nil)
		if err != nil {
			traceLog(fmt.Sprintf("restapi get creation error: %v", err))
			createMessage("restapi.creation.event", fmt.Sprintf("%v", err), "")
		}

		// Füge den Authorization-Header zum Request hinzu
		req.Header.Add("Authorization", "Bearer "+token)

		// Erstelle einen neuen HTTP-Client und führe den Request aus
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi get processing error: %v", err))
			createMessage("restapi.processing.error.event", fmt.Sprintf("%v", err), "")

		} else {

			// Lies den Response Body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				traceLog(fmt.Sprintf("restapi get error reading response: %v", err))
				createMessage("restapi.get.error.event", fmt.Sprintf("%v", err), "")

			} else {
				// Gib den Response Body aus
				debugLog(5, fmt.Sprintf("restapi get received response: %v", string(body)))
				rout <- string(body)
			}

			resp.Body.Close()
		}
	}
}

func restApiPost(rin chan Requestin) {
	url := genVar.Resturl     // Die URL der API, die du aufrufen möchtest
	token := genVar.Resttoken // Der Bearer Token für die Authentifizierung

	for {
		request := <-rin
		requrl := url + "/" + request.Node + "/" + request.Item
		data := request.Data
		// Erstelle einen neuen Request
		req, err := http.NewRequest("POST", requrl, strings.NewReader(data))
		if err != nil {
			traceLog(fmt.Sprintf("restapi post creation error: %v", err))
			createMessage("restapi.creation.event", fmt.Sprintf("%v", err), "")
		}

		req.Header.Set("Content-Type", "text/plain")
		// Füge den Authorization-Header zum Request hinzu
		req.Header.Add("Authorization", "Bearer "+token)

		// Erstelle einen neuen HTTP-Client und führe den Request aus
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi post processing error: %v", err))
			createMessage("restapi.processing.event", fmt.Sprintf("%v", err), "")
		}

		// Prüfe den Statuscode des Response, um sicherzustellen, dass der Request erfolgreich war
		if resp.StatusCode != http.StatusOK && resp.StatusCode != 202 {
			traceLog(fmt.Sprintf("restapi post statuscode: %d", resp.StatusCode))
			createMessage("restapi.status.event", fmt.Sprintf("%d", resp.StatusCode), "")
		}
		resp.Body.Close()
	}
}
