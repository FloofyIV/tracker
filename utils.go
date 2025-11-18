package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
	"github.com/tidwall/gjson"
)

func getUniverseFromPlaceID(PlaceID string) string {
	var universeID string
	var fails int

	for {
		if fails >= 5 {
			fmt.Printf("Too many fails, trying again in 15 minutes. (%d fails)\n", fails)
			time.Sleep(15 * time.Minute)
		}

		client := &http.Client{}
		url := "https://apis.roblox.com/universes/v1/places/" + PlaceID + "/universe"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fails++
			fmt.Println("Failed to create new GET request, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			fails++
			fmt.Println("Failed to get response, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fails++
			fmt.Println("Failed to read body, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}

		universeID = gjson.GetBytes(body, "universeId").String()
		if universeID == "" {
			fails++
			fmt.Println("Failed to get universe ID, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}

		break
	}

	return universeID
}

func webhookSend(name, webhookURL, description, role string) error {
	var embed discordwebhook.Embed

	if description == "" {
		embed = discordwebhook.Embed{
			Title:     name,
			Color:     16768512,
			Timestamp: time.Now(),
			Author: discordwebhook.Author{
				Name:     "Aesthetical",
				Icon_URL: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
			},
			Fields: []discordwebhook.Field{
				{Value: "Update detected."},
			},
		}
	} else {
		embed = discordwebhook.Embed{
			Title:     name,
			Color:     16768512,
			Timestamp: time.Now(),
			Author: discordwebhook.Author{
				Name:     "Aesthetical",
				Icon_URL: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
			},
			Fields: []discordwebhook.Field{
				{
					Name:  "Description updated",
					Value: description,
				},
			},
		}
	}

	var hook discordwebhook.Hook
	if role == "" {
		hook = discordwebhook.Hook{
			Username:   "Aesthetical",
			Avatar_url: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
			Embeds:     []discordwebhook.Embed{embed},
		}
	} else {
		hook = discordwebhook.Hook{
			Username:   "Aesthetical",
			Avatar_url: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
			Content:    "<@&" + role + ">",
			Embeds:     []discordwebhook.Embed{embed},
		}
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		return err
	}

	return discordwebhook.ExecuteWebhook(webhookURL, payload)
}

func getUniverseData(gameID string) (gameData, error) {
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID
	resp, err := http.Get(url)
	if err != nil {
		return gameData{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return gameData{}, err
	}

	var game gameData
	err = json.Unmarshal(body, &game)
	return game, err
}
