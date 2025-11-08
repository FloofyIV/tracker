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

	"github.com/tidwall/gjson"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
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

func getUniverseFromPlaceID(PlaceID string) string {
	client := &http.Client{}
	url := "https://apis.roblox.com/universes/v1/places/" + PlaceID + "/universe"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:144.0) Gecko/20100101 Firefox/144.0")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	universeID := gjson.Get(string(body), "universeId").String()

	if universeID == "" {
		panic("failed to get universeID, quitting.")
	}
	return universeID
}

func updateLoopHandler(gameID string, webhookURL string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		err := UpdateLoop(gameID, webhookURL)
		if err != nil {
			log.Printf("Error in updateLoop: %v. Retrying in 30 seconds...", err)
			time.Sleep(30 * time.Second)
			continue
		}
	}
}

func UpdateLoop(gameID string, webhookURL string) error {
	var lastUpdate time.Time
	var currentUpdate time.Time
	var name string
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID
	fmt.Println("Started updateLoop")

	for {
		fmt.Printf("Sending request...\r")
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		fmt.Printf("\033[KRecieved data\r")

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		var game gameData
		err = json.Unmarshal(body, &game)
		if err != nil {
			return err
		}

		for _, item := range game.Data {
			currentUpdate = item.Updated
			name = item.Name
		}

		if lastUpdate.IsZero() {
			lastUpdate = currentUpdate
		} else {
			if currentUpdate != lastUpdate {
				fmt.Println("update detected", time.Now().UTC())
				embed := discordwebhook.Embed{
					Title:     name,
					Color:     16768512,
					Timestamp: time.Now(),
					Author: discordwebhook.Author{
						Name:     "Aesthetical",
						Icon_URL: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
					},
					Fields: []discordwebhook.Field{
						{
							Value: "Update detected.",
						},
					},
				}

				hook := discordwebhook.Hook{
					Username:   "Aesthetical",
					Avatar_url: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
					Content:    "",
					Embeds:     []discordwebhook.Embed{embed},
				}

				payload, err := json.Marshal(hook)
				if err != nil {
					return err
				}

				err = discordwebhook.ExecuteWebhook(webhookURL, payload)
				if err != nil {
					return err
				}

				lastUpdate = currentUpdate
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func main() {
	webhookURL := os.Getenv("WEBHOOK")
	placeID := os.Getenv("PLACE")
	var wg sync.WaitGroup

	if webhookURL == "" {
		log.Fatal("please set the webhook (WEBHOOK=\"discord.com/xxx\" ./tracker)")
	} else if placeID == "" {
		log.Fatal("please set the placeID (PLACE=\"123456789\" ./tracker)")
	}
	fmt.Printf("\033[KGetting universeID\r")
	universeID := getUniverseFromPlaceID(placeID)
	fmt.Printf("\033[KGot universeID\n")
	wg.Add(1)
	go updateLoopHandler(universeID, webhookURL, &wg)
	wg.Wait()
}
