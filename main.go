package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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

func updateLoop(gameID string, webhookURL string, wg *sync.WaitGroup) {
	var lastUpdate time.Time
	var currentUpdate time.Time
	var name string
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID

	for {
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var game gameData
		err = json.Unmarshal(body, &game)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range game.Data {
			currentUpdate = item.Updated
			name = item.Name
		}
		if lastUpdate.IsZero() {
			lastUpdate = currentUpdate
		} else {
			if currentUpdate != lastUpdate {
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
					log.Fatal(err)
				}
				err = discordwebhook.ExecuteWebhook(webhookURL, payload)
				if err != nil {
					log.Fatal(err)
				}
				lastUpdate = currentUpdate
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func main() {
	webhookURL := os.Getenv("WEBHOOK")
	gameID := os.Getenv("GAME")
	if webhookURL == "" {
		log.Fatal("please set WEBHOOK (WEBHOOK=\"discord.com/xxx\" ./tracker)")
	} else if gameID == "" {
		log.Fatal("please set GAMEIF (GAMEID=\"123456789\" ./tracker)")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go updateLoop(gameID, webhookURL, &wg)
	wg.Wait()
}
