package openproject_webhooks_bot

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

func LoadUsersConfig(path string) map[string]int64 {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read users config %s: %v", path, err)
	}

	var users map[string]int64
	if err := json.Unmarshal(data, &users); err != nil {
		log.Fatalf("Failed to parse users config %s: %v", path, err)
	}

	log.Printf("Loaded %d user(s) from config", len(users))
	return users
}
