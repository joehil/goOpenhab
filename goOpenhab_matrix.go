package main

import (
        "log"

        "github.com/matrix-org/gomatrix"
)

func matrixSend(message string) {
        client, err := gomatrix.NewClient(genVar.matrix_homeserver, "", "")
        if err != nil {
                log.Printf("Matrix: Fehler beim Erstellen des Clients: %v", err)
        }

        resp, err := client.Login(&gomatrix.ReqLogin{
                Type:     "m.login.password",
                User:     genVar.matrix_username,
                Password: genVar.matrix_password,
        })
        if err != nil {
                log.Printf("Matrix: Login fehlgeschlagen: %v", err)
        }
        client.SetCredentials(resp.UserID, resp.AccessToken)

        _, err = client.SendText(genVar.matrix_roomID, message)
        if err != nil {
                log.Printf("Matrix: Fehler beim Senden: %v", err)
        }
}

