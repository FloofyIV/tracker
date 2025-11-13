package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func updateLoop(gameID string, webhookURL string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Starting update loop.")
	for {
		err := update(gameID, webhookURL)
		if err != nil {
			fmt.Println("retrying in 30 seconds, ", err)
			time.Sleep(30 * time.Second)
			continue
		}
		time.Sleep(30 * time.Second)
	}
}

func update(gameID string, webhookURL string) error {
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID // game url
	fmt.Printf("Sending request...\r")
	resp, err := http.Get(url) // http.Get() the game url -> resp
	if err != nil {
		return err
	}
	fmt.Printf("\033[KRecieved data, %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body) // extract the body from the response
	resp.Body.Close()
	if err != nil {
		return err
	}

	var game gameData
	err = json.Unmarshal(body, &game) // extract the json data from the body -> game
	if err != nil {
		return err
	}

	for _, item := range game.Data { // iterate through every key in the json body, saving them to variables
		currentUpdate = item.Updated.UTC()
		name = item.Name
	}

	if lastUpdate.IsZero() {
		lastUpdate = currentUpdate // if lastUpdate is empty, make it equal to current update
		fmt.Println("lastUpdate <- currentUpdate")
	} else {
		fmt.Println("current update: " + currentUpdate.Format(time.RFC850))
		fmt.Println("last update: " + lastUpdate.Format(time.RFC850))
		fmt.Println("name: " + name)
		if currentUpdate.After(lastUpdate) { // if the current update time is later than the last update, run this
			fmt.Println("update detected", time.Now().UTC())
			if webhookURL != "" {
				for i := 0; i < 3; i++ {
					err = webhookSend(name, webhookURL)
					if err == nil {
						break
					} else {
						fmt.Println(err)
					}
				}
			}
			lastUpdate = currentUpdate
		}
	}

	return nil
}

func main() {
	webhookURL := os.Getenv("WEBHOOK")
	placeID := os.Getenv("PLACE")
	var wg sync.WaitGroup

	if webhookURL == "" {
		fmt.Println("running with no webhook, set with PLACE=\"https://discord.com/api/webhook/xxx/xxx\"")
	} else if placeID == "" {
		log.Fatal("please set the placeID (PLACE=\"123456789\" ./tracker)")
	}
	fmt.Printf("\033[KGetting universeID\r")
	universeID := getUniverseFromPlaceID(placeID)
	fmt.Printf("\033[KGot universeID\n")
	wg.Add(1)
	go updateLoop(universeID, webhookURL, &wg)
	wg.Wait()
}
