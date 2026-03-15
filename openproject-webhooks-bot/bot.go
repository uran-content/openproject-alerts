package openproject_webhooks_bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	botAPI      *tgbotapi.BotAPI
	usersConfig map[string]int64
	mainChatID  int64
)

func Init(configPath string) {
	var err error

	botAPI, err = tgbotapi.NewBotAPI(os.Getenv("OP_WEBHOOKS_BOT_TG_BOT_KEY"))
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}
	log.Printf("Authorized on Telegram bot account %s", botAPI.Self.UserName)

	mainChatID, _ = strconv.ParseInt(os.Getenv("OP_WEBHOOKS_BOT_TG_MAIN_CHAT_ID"), 10, 64)

	usersConfig = LoadUsersConfig(configPath)
}

func StartBotListener() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			chatID := update.Message.Chat.ID
			text := fmt.Sprintf(
				"Привет! Ваш Telegram ID: <code>%d</code>\n\n"+
					"Отправьте этот ID разработчику, чтобы получать уведомления о задачах.",
				chatID,
			)
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "HTML"
			if _, err := botAPI.Send(msg); err != nil {
				log.Printf("Failed to send /start reply: %v", err)
			}
		}
	}
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		return
	}

	b, _ := ioutil.ReadAll(r.Body)
	log.Println(string(b))

	var g workPackage

	err = json.Unmarshal(b, &g)
	if err != nil {
		log.Println(err)
		return
	}

	var templateFile string
	switch r.URL.Path {
	case "/wp/created":
		templateFile = "message_work_package.html"
	case "/wp/updated":
		templateFile = "message_wp_updated.html"
	default:
		return
	}

	text := renderTemplate(templateFile, &g)

	// Send to main group chat
	if mainChatID != 0 {
		sendMessage(mainChatID, text)
	}

	// Send personal notifications to assignee and responsible
	notified := make(map[int64]bool)

	if g.WorkPackage.Embedded.Assignee != nil && g.WorkPackage.Embedded.Assignee.Email != "" {
		if chatID, ok := usersConfig[g.WorkPackage.Embedded.Assignee.Email]; ok {
			sendMessage(chatID, text)
			notified[chatID] = true
		}
	}

	if g.WorkPackage.Embedded.Responsible != nil && g.WorkPackage.Embedded.Responsible.Email != "" {
		if chatID, ok := usersConfig[g.WorkPackage.Embedded.Responsible.Email]; ok {
			if !notified[chatID] {
				sendMessage(chatID, text)
			}
		}
	}
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if _, err := botAPI.Send(msg); err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
	}
}

func renderTemplate(templateFile string, w *workPackage) string {
	var b bytes.Buffer

	tmpl, err := template.ParseFiles("openproject-webhooks-bot/templates/" + templateFile)
	if err != nil {
		log.Printf("Failed to parse template %s: %v", templateFile, err)
		return ""
	}
	if err := tmpl.Execute(&b, w); err != nil {
		log.Printf("Failed to execute template %s: %v", templateFile, err)
		return ""
	}

	return b.String()
}
