package main

import (
	"log"
	"net/http"

	bot "go-openproject-webhooks-bot/openproject-webhooks-bot"
)

func main() {
	bot.Init("config/users.json")
	go bot.StartBotListener()

	http.HandleFunc("/wp/created", bot.ServeHTTP)
	http.HandleFunc("/wp/updated", bot.ServeHTTP)

	log.Println("Starting HTTPS server on :443")
	err := http.ListenAndServeTLS(":443", "/srv/app/certs/server.crt", "/srv/app/certs/server.key", nil)
	if err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}
}
