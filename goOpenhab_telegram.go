package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

func sendTelegram(msg chan string) {
	for {
		//		message := <-msg
		bot, err := tgbotapi.NewBotAPI(genVar.tbtoken)
		if err != nil {
			fmt.Printf("Telegram error: %v\n", err)
			return
		}
		m := tgbotapi.NewMessage(genVar.chatid, <-msg)
		bot.Send(m)
	}
}
