package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
)

func getUniverseFromPlaceID(PlaceID string) string {
	var universeID string
	client := &http.Client{}

	backoff := time.Second * 2
	maxRetries := 7

	for attempt := 1; attempt <= maxRetries; attempt++ {
		url := "https://apis.roblox.com/universes/v1/places/" + PlaceID + "/universe"

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// i hate rate limittt
		if resp.StatusCode == 429 {
			retryAfter := time.Second * 10
			if v := resp.Header.Get("Retry-After"); v != "" {
				if dur, err := time.ParseDuration(v + "s"); err == nil {
					retryAfter = dur
				}
			}
			resp.Body.Close()
			time.Sleep(retryAfter)
			continue
		}

		// retry on 5xx
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		universeID = gjson.GetBytes(body, "universeId").String()
		if universeID != "" {
			return universeID
		}

		time.Sleep(backoff)
		backoff *= 2
	}

	fmt.Println("Failed to get universeID")
	return ""
}

func webhookSend(name, webhookURL, lastDescription, currentDescription, role string) error {
	type EmbedField struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	type Embed struct {
		Title     string       `json:"title,omitempty"`
		Color     int          `json:"color,omitempty"`
		Timestamp string       `json:"timestamp,omitempty"`
		Author    interface{}  `json:"author,omitempty"`
		Fields    []EmbedField `json:"fields,omitempty"`
	}

	type Payload struct {
		Username  string  `json:"username,omitempty"`
		AvatarURL string  `json:"avatar_url,omitempty"`
		Content   string  `json:"content,omitempty"`
		Embeds    []Embed `json:"embeds,omitempty"`
	}

	embed := Embed{
		Title:     name,
		Color:     16768512,
		Timestamp: time.Now().Format(time.RFC3339),
		Author: map[string]string{
			"name":     "Aesthetical",
			"icon_url": "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
		},
		Fields: []EmbedField{},
	}

	if currentDescription == lastDescription {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:  "Status",
			Value: "Update detected.",
		})
	} else {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:  "Description updated",
			Value: currentDescription,
		})
	}

	payload := Payload{
		Username:  "Aesthetical",
		AvatarURL: "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024",
		Embeds:    []Embed{embed},
	}

	if role != "" {
		payload.Content = "<@&" + role + ">"
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("request build error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request send error: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	log.Printf("Webhook Response Status: %d", resp.StatusCode)
	log.Printf("Webhook Response Body: %s", string(respBody))

	if resp.StatusCode == 429 {
		retryAfter := 5 * time.Second
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if dur, err := time.ParseDuration(ra + "s"); err == nil {
				retryAfter = dur
			}
		}
		log.Printf("Rate limited by Discord. Retrying after %v", retryAfter)
		time.Sleep(retryAfter)
		return fmt.Errorf("rate limited")
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("discord returned %d: %s", resp.StatusCode, string(respBody))
	}

	log.Println("Webhook sent successfully.")
	return nil
}

func getUniverseData(gameID string) (gameData, error) {
	url := "https://games.roblox.com/v1/games?universeIds=" + gameID
	maxRetries := 6
	backoff := time.Second * 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// handle 429 rate limit
		if resp.StatusCode == 429 {
			log.Printf("rate limit")
			retryAfter := time.Second * 10
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if dur, err := time.ParseDuration(ra + "s"); err == nil {
					retryAfter = dur
				}
			}
			resp.Body.Close()
			time.Sleep(retryAfter)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return gameData{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		var game gameData
		err = json.Unmarshal(body, &game)
		if err != nil {
			return gameData{}, err
		}

		return game, nil
	}

	return gameData{}, fmt.Errorf("getUniverseData failed, max retries")
}
