package openproject_webhooks_bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	botAPI      *tgbotapi.BotAPI
	usersConfig map[string]int64
	mainChatID  int64
	allowedIP   string
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

	allowedIP = os.Getenv("OP_WEBHOOKS_BOT_ALLOWED_IP")
	if allowedIP != "" {
		log.Printf("Webhook IP filter enabled: only accepting from %s", allowedIP)
	} else {
		log.Println("WARNING: OP_WEBHOOKS_BOT_ALLOWED_IP not set, accepting webhooks from any IP")
	}
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
	if allowedIP != "" {
		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteIP = r.RemoteAddr
		}
		if remoteIP != allowedIP {
			log.Printf("Rejected webhook from unauthorized IP: %s", remoteIP)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

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
	taskRef := fmt.Sprintf("#%d %s", g.WorkPackage.ID, g.WorkPackage.Subject)

	// Send personal notifications to assignee and responsible
	notified := make(map[int64]bool)

	if g.WorkPackage.Embedded.Assignee != nil && g.WorkPackage.Embedded.Assignee.Email != "" {
		email := g.WorkPackage.Embedded.Assignee.Email
		if chatID, ok := usersConfig[email]; ok {
			notifyUser(chatID, email, "assignee", taskRef, text)
			notified[chatID] = true
		} else {
			sendLogMessage(fmt.Sprintf("⚠️ Assignee %s не найден в конфиге, уведомление не отправлено (таск %s)", email, taskRef))
		}
	}

	if g.WorkPackage.Embedded.Responsible != nil && g.WorkPackage.Embedded.Responsible.Email != "" {
		email := g.WorkPackage.Embedded.Responsible.Email
		if chatID, ok := usersConfig[email]; ok {
			if !notified[chatID] {
				notifyUser(chatID, email, "responsible", taskRef, text)
			}
		} else {
			sendLogMessage(fmt.Sprintf("⚠️ Responsible %s не найден в конфиге, уведомление не отправлено (таск %s)", email, taskRef))
		}
	}
}

func notifyUser(chatID int64, email, role, taskRef, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if _, err := botAPI.Send(msg); err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
		sendLogMessage(fmt.Sprintf("❌ Не удалось отправить уведомление %s (%s, %s): %v", email, role, taskRef, err))
	} else {
		sendLogMessage(fmt.Sprintf("✅ Уведомление отправлено %s (%s) — %s", email, role, taskRef))
	}
}

func sendLogMessage(text string) {
	if mainChatID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(mainChatID, text)
	msg.ParseMode = "HTML"
	if _, err := botAPI.Send(msg); err != nil {
		log.Printf("Failed to send log message to main chat: %v", err)
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
