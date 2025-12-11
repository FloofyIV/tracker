package main

import (
	"fmt"
	"log"
	"net/http"
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
	LogFile.WriteString("Starting update loop.\n")

	var lastUpdate time.Time
	var lastDescription string

	for {
		data, err := getUniverseData(gameID)
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC850), err)
			time.Sleep(30 * time.Second)
			continue
		}

		item := data.Data[0]

		currentUpdate := item.Updated
		currentDescription := item.Description
		name := item.Name

		if lastUpdate.IsZero() {
			lastUpdate = currentUpdate
			lastDescription = currentDescription

			time.Sleep(30 * time.Second)
			continue
		}

		updateChanged := currentUpdate.After(lastUpdate)
		descChanged := currentDescription != lastDescription

		if updateChanged || descChanged {
			fmt.Println("Update detected", time.Now().UTC())

			if webhookURL != "" {
				for i := 0; i < 3; i++ {
					err := webhookSend(name, webhookURL, lastDescription, currentDescription, role)
					if err == nil {
						break
					}
					fmt.Println(err)
					time.Sleep(2 * time.Second)
				}
			}

			lastUpdate = currentUpdate
			lastDescription = currentDescription

			time.Sleep(30 * time.Second)
			continue
		}

		time.Sleep(30 * time.Second)
	}
}

func main() {
	var err error

	webhookURL := os.Getenv("WEBHOOK")
	placeID := os.Getenv("PLACE")
	pingRole := os.Getenv("ROLE")

	LogFile, err = os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0655)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Tracker started, %s\n", time.Now().Format(time.RFC850))
	fmt.Fprintf(LogFile, "Tracker started, %s\n", time.Now().Format(time.RFC850))

	if webhookURL == "" {
		fmt.Printf("Running with no webhook.\n")
		fmt.Fprintf(LogFile, "Running with no webhook.\n")
	} else if webhookURL[:33] == "https://discord.com/api/webhooks/" && len(webhookURL) == 121 {
		fmt.Printf("Testing webhook\n")
		fmt.Fprintf(LogFile, "Testing webhook\n")
		resp, err := http.Get(webhookURL)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			panic(resp.StatusCode)
		}
		fmt.Printf("Webhook Verified\n")
	}
	if placeID == "" {
		log.Fatal("Please set PLACE env var")
	}

	places := strings.Split(placeID, ";")
	var wg sync.WaitGroup

	for _, p := range places {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		fmt.Printf("Getting universe for place: %s\n", p)
		fmt.Fprintf(LogFile, "Getting universeID from placeID: %s\n", p)
		universeID := getUniverseFromPlaceID(p)
		fmt.Printf("Got universeID: %s\n", universeID)
		fmt.Fprintf(LogFile, "Got UniverseID: %s\n", universeID)
		data, err := getUniverseData(universeID)
		if err != nil {
			panic(err)
		}
		item := data.Data[0]
		name := item.Name
		time.Sleep(30 * time.Second)
		wg.Add(1)
		go mainLoop(universeID, webhookURL, pingRole, &wg)
		fmt.Printf("Tracking %s\n", name)
		fmt.Fprintf(LogFile, "Tracking %s\n", name)
	}
	wg.Wait()
}
