package main

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type gameData struct {
	Data []struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Updated     time.Time `json:"updated"`
	} `json:"data"`
}

var LogFile *os.File

func mainLoop(gameID, webhookURL, role string, wg *sync.WaitGroup) {
	defer wg.Done()

	var lastUpdate time.Time
	var lastDescription string

	for {
		data, err := getUniverseData(gameID)
		if err != nil {
			log.Printf("Error getting universe data: %v\n", err)
			time.Sleep(20 * time.Second)
			continue
		}

		if len(data.Data) == 0 {
			log.Println("Universe API returned empty data, retrying…")
			time.Sleep(20 * time.Second)
			continue
		}

		item := data.Data[0]
		currentUpdate := item.Updated
		currentDescription := item.Description
		name := item.Name

		if lastUpdate.IsZero() {
			lastUpdate = currentUpdate
			lastDescription = currentDescription
			time.Sleep(20 * time.Second)
			continue
		}

		updateChanged := currentUpdate.After(lastUpdate)
		descChanged := currentDescription != lastDescription

		if updateChanged || descChanged {
			log.Printf("Update detected at %s\n", time.Now().UTC())

			if webhookURL != "" {
				maxRetries := 3

				for attempt := 1; attempt <= maxRetries; attempt++ {
					err := webhookSend(name, webhookURL, lastDescription, currentDescription, role)
					if err == nil {
						log.Println("Webhook delivered")
						break
					}

					log.Printf("Webhook failed (attempt %d/%d): %v", attempt, maxRetries, err)
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
				}
			}

			lastUpdate = currentUpdate
			lastDescription = currentDescription
		}

		time.Sleep(20 * time.Second)
	}
}

func main() {
	webhookURL := os.Getenv("WEBHOOK")
	placeID := os.Getenv("PLACE")
	pingRole := os.Getenv("ROLE")

	if placeID == "" {
		log.Fatal("Missing PLACE environment variable")
	}

	places := strings.Split(placeID, ";")
	var wg sync.WaitGroup

	for _, p := range places {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		log.Printf("Resolving universe for place: %s\n", p)
		universeID := getUniverseFromPlaceID(p)

		if universeID == "" {
			log.Fatalf("Could not resolve Universe ID for place %s", p)
		}

		log.Printf("Tracking universe %s", universeID)
		wg.Add(1)
		go mainLoop(universeID, webhookURL, pingRole, &wg)

		time.Sleep(2 * time.Second)
	}

	wg.Wait()
}
