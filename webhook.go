package main

import (
	"encoding/json"
	"time"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
)

func webhookSend(name string, webhookURL string) error {
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
	return err
}
