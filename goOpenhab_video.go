package main

import (
	"log"
	"net/http"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const videoOutputDir = "/opt/homeautomation/video/"

func recordVideo(camera string, duration string) {
	now := time.Now()
	name := now.Format("2006-01-02_15-04-05")

	resp, err := http.Get(camera)
	if err != nil {
		log.Println("Fehler beim Abrufen des Videostreams:", err)
		return
	} else {
		resp.Body.Close()
	}

	err = ffmpeg.Input(camera, ffmpeg.KwArgs{}).
		Output(videoOutputDir+"video"+name+".mp4", ffmpeg.KwArgs{"t": duration}).
		OverWriteOutput().ErrorToStdOut().Run()

	if err != nil {
		log.Printf("Error recording video '%s': %v", name, err)
	} else {
		log.Printf("Video recorded successfully: %s", name)
	}
}
