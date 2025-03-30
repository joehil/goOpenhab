package main

import (
	"log"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func recordVideo(camera string, duration string) {
	now := time.Now()
	name := now.Format("2006-01-02_15-04-05")

	err := ffmpeg.Input(camera, ffmpeg.KwArgs{}).
		Output("/opt/homeautomation/video/video"+name+".mp4", ffmpeg.KwArgs{"t": duration}).
		OverWriteOutput().ErrorToStdOut().Run()
	log.Println("Video recorded: "+name, err)
}
