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
		}

		// Füge den Authorization-Header zum Request hinzu
		req.Header.Add("Authorization", "Bearer "+token)

		// Erstelle einen neuen HTTP-Client und führe den Request aus
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi get processing error: %v", err))
		}

		// Lies den Response Body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			traceLog(fmt.Sprintf("restapi get error reading response: %v", err))
		} else {
			// Gib den Response Body aus
			msgLog(fmt.Sprintf("restapi get received response: %v", string(body)))
			rout <- string(body)
		}

		resp.Body.Close()
	}
}

func restApiPut(rin chan Requestin) {
	url := genVar.Resturl     // Die URL der API, die du aufrufen möchtest
	token := genVar.Resttoken // Der Bearer Token für die Authentifizierung

	for {
		request := <-rin
		requrl := url + "/" + request.Node + "/" + request.Item + "/" + request.Value
		data := request.Data
		// Erstelle einen neuen Request
		req, err := http.NewRequest("PUT", requrl, strings.NewReader(data))
		if err != nil {
			traceLog(fmt.Sprintf("restapi put creation error: %v", err))
		}

		req.Header.Set("Content-Type", "text/plain")
		// Füge den Authorization-Header zum Request hinzu
		req.Header.Add("Authorization", "Bearer "+token)

		// Erstelle einen neuen HTTP-Client und führe den Request aus
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi put processing error: %v", err))
		}

		// Prüfe den Statuscode des Response, um sicherzustellen, dass der Request erfolgreich war
		if resp.StatusCode != http.StatusOK && resp.StatusCode != 202 {
			traceLog(fmt.Sprintf("restapi put statuscode: %v", resp.StatusCode))
		}
		resp.Body.Close()
	}
}
