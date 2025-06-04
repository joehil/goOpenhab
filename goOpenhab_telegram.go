package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

func sanitizeMessage(msg string) string {
	// Implement sanitization logic as needed
	return msg
}

func sendTelegram(msg chan string) {
	bot, err := tgbotapi.NewBotAPI(genVar.Tbtoken)
	if err != nil {
		fmt.Printf("Telegram error: %v\n", err)
		return
	}
	for {
		select {
		case rawMsg := <-msg:
			sanitizedMsg := sanitizeMessage(rawMsg)
			if len(sanitizedMsg) == 0 || len(sanitizedMsg) > 4096 {
				continue // skip empty or too long messages
			}
			m := tgbotapi.NewMessage(genVar.Chatid, sanitizedMsg)
			if _, err := bot.Send(m); err != nil {
				fmt.Printf("Failed to send Telegram message: %v\n", err)
				bot, err = tgbotapi.NewBotAPI(genVar.Tbtoken)
				if err != nil {
					fmt.Printf("Telegram error: %v\n", err)
				}
				bot.Send(m)
			}
		}
	}
}
