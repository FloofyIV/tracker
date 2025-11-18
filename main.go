package main

import (
	"fmt"
	"log"
	"os"
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
			fmt.Println("retrying in 30 seconds,", err)
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

		if currentDescription == lastDescription && currentUpdate.Equal(lastUpdate) {
			time.Sleep(30 * time.Second)
			continue
		}

		if currentUpdate.After(lastUpdate) {

			if currentDescription != lastDescription {

				fmt.Println("Description updated", time.Now().Format(time.RFC850))
				fmt.Fprintf(LogFile, "Description updated, %s\n", time.Now().Format(time.RFC850))

				if webhookURL != "" {
					desc := currentDescription
					for i := 0; i < 3; i++ {
						err := webhookSend(name, webhookURL, desc, role)
						if err == nil {
							break
						}
						fmt.Println(err)
						time.Sleep(2 * time.Second)
					}
				}

				lastDescription = currentDescription
				lastUpdate = currentUpdate
				time.Sleep(30 * time.Second)
				continue
			}

			fmt.Println("Update detected", time.Now().UTC())
			if webhookURL != "" {
				for i := 0; i < 3; i++ {
					err := webhookSend(name, webhookURL, "", role)
					if err == nil {
						break
					}
					fmt.Println(err)
					time.Sleep(2 * time.Second)
				}
			}

			lastUpdate = currentUpdate
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
	fmt.Fprintf(LogFile, "Tracker started, %s\n", time.Now().Format(time.RFC850))

	if webhookURL == "" {
		fmt.Println("Running with no webhook.")
	}
	if placeID == "" {
		log.Fatal("Please set PLACE env var")
	}

	fmt.Printf("Getting universeID...\n")
	universeID := getUniverseFromPlaceID(placeID)
	fmt.Printf("Got universeID\n")
	LogFile.WriteString("Got UniverseID\n")

	var wg sync.WaitGroup
	wg.Add(1)
	go mainLoop(universeID, webhookURL, pingRole, &wg)
	wg.Wait()
}
