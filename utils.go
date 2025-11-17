package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
	"github.com/tidwall/gjson"
)

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

func webhookSend(name string, webhookURL string, description string, role string) error {
	var embed discordwebhook.Embed
	if description != "" {
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
