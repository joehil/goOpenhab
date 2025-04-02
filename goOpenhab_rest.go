package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

/*
restApiGet continuously listens for incoming requests on the 'rin' channel,
constructs a GET request to a specified REST API using the provided URL and
Bearer token, and sends the API's response body to the 'rout' channel. It
handles errors in request creation, execution, and response reading, logging
them and sending error messages as needed.
*/
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
		req.Header.Set("Authorization", "Bearer "+token)

		// Führe den Request aus
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi get processing error: %v", err))
			createMessage("restapi.processing.error.event", fmt.Sprintf("%v", err), "")
		} else if resp != nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					traceLog(fmt.Sprintf("error closing response body: %v", err))
				}
			}()

			// Lies den Response Body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				traceLog(fmt.Sprintf("restapi get error reading response: %v", err))
				createMessage("restapi.get.error.event", fmt.Sprintf("%v", err), "")
				genVar.Telegram <- "goOpenhab restapi error: " + fmt.Sprintf("%v", err)
			} else {
				// Gib den Response Body aus
				debugLog(5, fmt.Sprintf("restapi get received response: %v", string(body)))
				rout <- string(body)
			}
		}
	}
}

/*
restApiPost sends POST requests to a specified REST API endpoint.

This function continuously listens for incoming requests on the provided channel.
For each request, it constructs a URL using the base URL, node, and item from the request,
and sends a POST request with the request data. It includes a Bearer token for authentication
and sets the content type to "text/plain". The function logs any errors encountered during
request creation, execution, or response handling, and checks the response status code
to ensure the request was successful.
*/
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
		req.Header.Set("Authorization", "Bearer "+token)

		// Führe den Request aus
		resp, err := client.Do(req)
		if err != nil {
			traceLog(fmt.Sprintf("restapi post processing error: %v", err))
			createMessage("restapi.processing.event", fmt.Sprintf("%v", err), "")
		}

		// Prüfe den Statuscode des Response, um sicherzustellen, dass der Request erfolgreich war
		if resp != nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					traceLog(fmt.Sprintf("error closing response body: %v", err))
				}
			}()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != 202 {
				traceLog(fmt.Sprintf("restapi post statuscode: %d", resp.StatusCode))
				createMessage("restapi.status.event", fmt.Sprintf("%d", resp.StatusCode), "")
			}
		}
	}
}
