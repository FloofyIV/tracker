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
			fmt.Printf("Too many fails, trying again in 15 minutes. (%d fails)", fails)
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
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:144.0) Gecko/20100101 Firefox/144.0")

		resp, err := client.Do(req)
		if err != nil {
			fails++
			fmt.Println("Failed to get/send response/request, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fails++
			fmt.Println("Failed to read the body, retrying in 30 seconds")
			time.Sleep(30 * time.Second)
			continue
		}
		universeID = gjson.Get(string(body), "universeId").String()

		if universeID == "" {
			fails++
			fmt.Println("Failed to get universe id, retrying in 30 seconds.")
			time.Sleep(30 * time.Second)
			continue
		}
		break
	}
	return universeID
}

func webhookSend(name string, webhookURL string, description string, role string) error {
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
				{
					Value: "Update detected.",
				},
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
					Value: currentDescription,
				},
			},
		}
	}
	var hook discordwebhook.Hook
	if role == "" {
		hook = discordwebhook.Hook{
			Username:   "Aesthetical",
			Avatar_url: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
			Content:    "",
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

	err = discordwebhook.ExecuteWebhook(webhookURL, payload)
	if err != nil {
		return err
	}
	return err
}

func getUniverseData(gameID string) (gameData, error) {
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID // game url
	fmt.Printf("Sending request...\r")
	resp, err := http.Get(url) // http.Get() the game url -> resp
	if err != nil {
		return gameData{}, err
	}
	fmt.Printf("\033[KRecieved data, %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body) // extract the body from the response
	resp.Body.Close()
	if err != nil {
		return gameData{}, err
	}

	var game gameData
	err = json.Unmarshal(body, &game) // extract the json data from the body -> game
	if err != nil {
		return gameData{}, err
	}

	return game, err
}
