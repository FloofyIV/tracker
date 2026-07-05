package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	botName      = "Aesthetical Tracker"
	botAvatarURL = "https://cdn.discordapp.com/avatars/1419099472650043555/c11c5e3a7e55d7adc756f47a956eb6fb.webp?size=1024"

	colorGameUpdate   = 0xFFC000 // gold — a tracked game changed
	colorPlayerJoin   = 0x57F287 // green  — user started playing
	colorPlayerLeave  = 0x99AAB5 // grey   — user stopped playing
	colorPlayerOnline = 0x5865F2 // blurple — user came online (not in-game)
)

type embedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type embedAuthor struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
	URL     string `json:"url,omitempty"`
}

type embedThumbnail struct {
	URL string `json:"url"`
}

type embedFooter struct {
	Text string `json:"text"`
}

type embed struct {
	Title     string          `json:"title,omitempty"`
	URL       string          `json:"url,omitempty"`
	Color     int             `json:"color,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
	Author    *embedAuthor    `json:"author,omitempty"`
	Thumbnail *embedThumbnail `json:"thumbnail,omitempty"`
	Fields    []embedField    `json:"fields,omitempty"`
	Footer    *embedFooter    `json:"footer,omitempty"`
}

type webhookPayload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []embed `json:"embeds,omitempty"`
}

type Discord struct {
	http *http.Client
}

func newDiscord(timeout time.Duration) *Discord {
	return &Discord{http: &http.Client{Timeout: timeout}}
}

func (d *Discord) send(ctx context.Context, webhookURL, role string, e embed) error {
	if webhookURL == "" {
		return nil
	}
	payload := webhookPayload{
		Username:  botName,
		AvatarURL: botAvatarURL,
		Embeds:    []embed{e},
	}
	if role != "" {
		payload.Content = "<@&" + role + ">"
	}

	res, err := fetchWithRetry(ctx, d.http, newJSONPost(webhookURL, payload), 4)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	if res.Status < 200 || res.Status > 299 {
		return fmt.Errorf("discord returned %d: %s", res.Status, string(res.Body))
	}
	return nil
}

func numFmt(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// maxDiffFieldChars leaves room for both an "Old" and "New" block plus
// labels/formatting within Discord's 1024-char embed field value limit.
const maxDiffFieldChars = 460

func gameEmbed(game GameInfo, placeID, iconURL string, c *gameChange) embed {
	color := colorGameUpdate
	if !c.Significant {
		color = colorPlayerLeave // muted grey for non-visible/metadata-only refreshes
	}

	e := embed{
		Title:     game.Name,
		URL:       "https://www.roblox.com/games/" + placeID,
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Author:    &embedAuthor{Name: botName, IconURL: botAvatarURL},
		Fields: []embedField{
			{Name: "Status", Value: c.Status},
			{Name: "Players Online", Value: numFmt(game.Playing), Inline: true},
			{Name: "Total Visits", Value: numFmt(game.Visits), Inline: true},
			{Name: "Favorites", Value: numFmt(game.FavoritedCount), Inline: true},
		},
	}

	if c.NameChanged {
		e.Fields = append(e.Fields, embedField{
			Name:  "📝 Name Changed",
			Value: fmt.Sprintf("**Old:** %s\n**New:** %s", truncate(c.OldName, maxDiffFieldChars), truncate(c.NewName, maxDiffFieldChars)),
		})
	}

	if c.DescChanged {
		oldDesc := c.OldDesc
		if oldDesc == "" {
			oldDesc = "*(empty)*"
		}
		newDesc := c.NewDesc
		if newDesc == "" {
			newDesc = "*(empty)*"
		}
		e.Fields = append(e.Fields, embedField{
			Name:  "📄 Description Changed — Before",
			Value: truncate(oldDesc, maxDiffFieldChars),
		})
		e.Fields = append(e.Fields, embedField{
			Name:  "📄 Description Changed — After",
			Value: truncate(newDesc, maxDiffFieldChars),
		})
	}

	if c.UpdatedChanged {
		oldTS := "unknown"
		if !c.OldUpdated.IsZero() {
			oldTS = fmt.Sprintf("<t:%d:R>", c.OldUpdated.Unix())
		}
		newTS := fmt.Sprintf("<t:%d:R>", c.NewUpdated.Unix())
		e.Fields = append(e.Fields, embedField{
			Name:  "🕒 Last Updated Timestamp",
			Value: fmt.Sprintf("**Old:** %s\n**New:** %s", oldTS, newTS),
		})
	}

	if !c.Significant {
		e.Fields = append(e.Fields, embedField{
			Name:  "ℹ️ Note",
			Value: "Roblox reported an updated timestamp, but the name and description are unchanged. This is usually a backend metadata refresh, not a real edit.",
		})
	}

	e.Footer = &embedFooter{Text: fmt.Sprintf("Universe ID: %d  •  Place ID: %s", game.ID, placeID)}
	if iconURL != "" {
		e.Thumbnail = &embedThumbnail{URL: iconURL}
	}
	return e
}

func playerEmbed(username string, userID int64, avatarURL string, kind string, gameName, placeID string) embed {
	var title string
	var color int
	fields := []embedField{
		{Name: "Link to profile:", Value: fmt.Sprintf("[%s](https://www.roblox.com/users/%d/profile)", username, userID), Inline: true},
	}

	switch kind {
	case "join":
		title = fmt.Sprintf("%s started playing %s", username, gameName)
		color = colorPlayerJoin
		if gameName != "" {
			fields = append(fields, embedField{
				Name:   "Game",
				Value:  fmt.Sprintf("[%s](https://www.roblox.com/games/%s)", gameName, placeID),
				Inline: true,
			})
		}
	case "leave":
		title = fmt.Sprintf("%s stopped playing %s", username, gameName)
		color = colorPlayerLeave
	case "online":
		title = fmt.Sprintf("%s came online", username)
		color = colorPlayerOnline
	case "offline":
		title = fmt.Sprintf("%s went offline", username)
		color = colorPlayerLeave
	default:
		title = fmt.Sprintf("%s's status changed", username)
		color = colorPlayerOnline
	}

	e := embed{
		Title:     title,
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields:    fields,
		Footer:    &embedFooter{Text: fmt.Sprintf("User ID: %d", userID)},
	}
	if avatarURL != "" {
		e.Thumbnail = &embedThumbnail{URL: avatarURL}
	}
	return e
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
