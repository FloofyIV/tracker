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
		ID                int    `json:"id"`
		RootPlaceID       int    `json:"rootPlaceId"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		SourceName        any    `json:"sourceName"`
		SourceDescription any    `json:"sourceDescription"`
		Creator           struct {
			ID               int    `json:"id"`
			Name             string `json:"name"`
			Type             string `json:"type"`
			IsRNVAccount     bool   `json:"isRNVAccount"`
			HasVerifiedBadge bool   `json:"hasVerifiedBadge"`
		} `json:"creator"`
		Price                     any       `json:"price"`
		AllowedGearGenres         []string  `json:"allowedGearGenres"`
		AllowedGearCategories     []any     `json:"allowedGearCategories"`
		IsGenreEnforced           bool      `json:"isGenreEnforced"`
		CopyingAllowed            bool      `json:"copyingAllowed"`
		Playing                   int       `json:"playing"`
		Visits                    int64     `json:"visits"`
		MaxPlayers                int       `json:"maxPlayers"`
		Created                   time.Time `json:"created"`
		Updated                   time.Time `json:"updated"`
		StudioAccessToApisAllowed bool      `json:"studioAccessToApisAllowed"`
		CreateVipServersAllowed   bool      `json:"createVipServersAllowed"`
		UniverseAvatarType        string    `json:"universeAvatarType"`
		Genre                     string    `json:"genre"`
		GenreL1                   string    `json:"genre_l1"`
		GenreL2                   string    `json:"genre_l2"`
		IsAllGenre                bool      `json:"isAllGenre"`
		IsFavoritedByUser         bool      `json:"isFavoritedByUser"`
		FavoritedCount            int       `json:"favoritedCount"`
	} `json:"data"`
}

var lastUpdate time.Time
var currentUpdate time.Time
var name string
var lastDescription string
var currentDescription string
var LogFile *os.File

// var wasDescription bool

func mainLoop(gameID string, webhookURL string, role string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Starting update loop.")
	LogFile.WriteString("Starting update loop.")
	for {
		data, err := getUniverseData(gameID)
		if err != nil {
			fmt.Println("retrying in 30 seconds, ", err)
			time.Sleep(30 * time.Second)
			continue
		}
		for _, item := range data.Data { // iterate through every key in the json body, saving them to variables
			currentUpdate = item.Updated
			name = item.Name
			currentDescription = item.Description
		}
		if lastDescription == "" {
			lastDescription = currentDescription
		} else {
			if currentDescription != lastDescription {
				fmt.Println("Description updated", time.Now().Format(time.RFC850))
				fmt.Fprintf(LogFile, "Description updated, %s", time.Now().Format(time.RFC850))
				if webhookURL != "" {
					for i := 0; i < 3; i++ {
						err = webhookSend(name, webhookURL, currentDescription, role)
						if err == nil {
							break
						} else {
							fmt.Println(err)
						}
					}
				}
				lastDescription = currentDescription
				lastUpdate = currentUpdate
				time.Sleep(30 * time.Second)
				continue
			}
		}
		if lastUpdate.IsZero() {
			lastUpdate = currentUpdate // if lastUpdate is empty, make it equal to current update
			fmt.Println("lastUpdate <- currentUpdate")
		} else {
			if currentUpdate.After(lastUpdate) { // if the current update time is later than the last update, run this
				fmt.Println("update detected", time.Now().UTC())
				if webhookURL != "" {
					for i := 0; i < 3; i++ {
						err = webhookSend(name, webhookURL, "", role) // try to send to the webhook 3 times
						if err == nil {
							break // break out of the 3 time send loop if it succeeds.
						} else {
							fmt.Println(err)
						}
					}
				}
				lastUpdate = currentUpdate
			}
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
		fmt.Println(LogFile, err)
		panic(err)
	}
	fmt.Fprintf(LogFile, "Tracker started, %s", time.Now().Format(time.RFC850))
	var wg sync.WaitGroup

	if webhookURL == "" {
		fmt.Println("running with no webhook, set with PLACE=\"https://discord.com/api/webhook/xxx/xxx\"")
	} else if placeID == "" {
		log.Fatal("please set the placeID (PLACE=\"123456789\" ./tracker)")
	}
	fmt.Printf("\033[KGetting universeID\r")
	universeID := getUniverseFromPlaceID(placeID)
	LogFile.WriteString("Got UniverseID")
	fmt.Printf("\033[KGot universeID\n")
	wg.Add(1)
	go mainLoop(universeID, webhookURL, pingRole, &wg)
	wg.Wait()
}
