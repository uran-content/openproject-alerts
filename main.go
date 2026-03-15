package main

import (
	"net/http"

	bot "go-openproject-webhooks-bot/openproject-webhooks-bot"
)

func main() {
	bot.Init("config/users.json")
	go bot.StartBotListener()

	http.HandleFunc("/wp/created", bot.ServeHTTP)
	http.HandleFunc("/wp/updated", bot.ServeHTTP)

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		return
	}
}
