package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RobloxClient struct {
	http *http.Client
}

func newRobloxClient(timeout time.Duration) *RobloxClient {
	return &RobloxClient{http: &http.Client{Timeout: timeout}}
}

func (c *RobloxClient) ResolveUniverseID(ctx context.Context, placeID string) (string, error) {
	target := "https://apis.roblox.com/universes/v1/places/" + url.PathEscape(placeID) + "/universe"
	res, err := fetchWithRetry(ctx, c.http, newGetRequest(target), 7)
	if err != nil {
		return "", err
	}
	if res.Status != http.StatusOK {
		return "", fmt.Errorf("universe lookup for place %s returned status %d", placeID, res.Status)
	}
	var parsed struct {
		UniverseID int64 `json:"universeId"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return "", fmt.Errorf("parsing universe lookup: %w", err)
	}
	if parsed.UniverseID == 0 {
		return "", fmt.Errorf("place %s has no universe (invalid place ID?)", placeID)
	}
	return strconv.FormatInt(parsed.UniverseID, 10), nil
}

type GameInfo struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Playing        int64     `json:"playing"`
	Visits         int64     `json:"visits"`
	MaxPlayers     int       `json:"maxPlayers"`
	FavoritedCount int64     `json:"favoritedCount"`
	Updated        time.Time `json:"updated"`
	Creator        struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"creator"`
}

func (c *RobloxClient) GetGamesInfo(ctx context.Context, universeIDs []string) (map[string]GameInfo, error) {
	if len(universeIDs) == 0 {
		return map[string]GameInfo{}, nil
	}
	target := "https://games.roblox.com/v1/games?universeIds=" + strings.Join(universeIDs, ",")
	res, err := fetchWithRetry(ctx, c.http, newGetRequest(target), 6)
	if err != nil {
		return nil, err
	}
	if res.Status != http.StatusOK {
		return nil, fmt.Errorf("games API returned status %d", res.Status)
	}
	var parsed struct {
		Data []GameInfo `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing games response: %w", err)
	}
	out := make(map[string]GameInfo, len(parsed.Data))
	for _, g := range parsed.Data {
		out[strconv.FormatInt(g.ID, 10)] = g
	}
	return out, nil
}

func (c *RobloxClient) GetGameIcons(ctx context.Context, universeIDs []string) map[string]string {
	out := map[string]string{}
	if len(universeIDs) == 0 {
		return out
	}
	target := "https://thumbnails.roblox.com/v1/games/icons?universeIds=" +
		strings.Join(universeIDs, ",") + "&size=512x512&format=Png&isCircular=false"
	res, err := fetchWithRetry(ctx, c.http, newGetRequest(target), 3)
	if err != nil || res.Status != http.StatusOK {
		return out
	}
	var parsed struct {
		Data []struct {
			TargetID int64  `json:"targetId"`
			ImageURL string `json:"imageUrl"`
			State    string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return out
	}
	for _, d := range parsed.Data {
		if d.State == "Completed" && d.ImageURL != "" {
			out[strconv.FormatInt(d.TargetID, 10)] = d.ImageURL
		}
	}
	return out
}

func (c *RobloxClient) ResolveUsernamesToIDs(ctx context.Context, handles []string) (map[string]int64, error) {
	out := map[string]int64{}
	var lookup []string
	for _, h := range handles {
		if id, err := strconv.ParseInt(h, 10, 64); err == nil {
			out[h] = id
			continue
		}
		lookup = append(lookup, h)
	}
	if len(lookup) == 0 {
		return out, nil
	}

	target := "https://users.roblox.com/v1/usernames/users"
	payload := map[string]any{
		"usernames":          lookup,
		"excludeBannedUsers": false,
	}
	res, err := fetchWithRetry(ctx, c.http, newJSONPost(target, payload), 5)
	if err != nil {
		return out, err
	}
	if res.Status != http.StatusOK {
		return out, fmt.Errorf("username lookup returned status %d", res.Status)
	}
	var parsed struct {
		Data []struct {
			ID                int64  `json:"id"`
			Name              string `json:"name"`
			RequestedUsername string `json:"requestedUsername"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return out, fmt.Errorf("parsing username lookup: %w", err)
	}
	for _, d := range parsed.Data {
		out[d.RequestedUsername] = d.ID
	}
	return out, nil
}

func (c *RobloxClient) GetAvatarHeadshots(ctx context.Context, userIDs []int64) map[int64]string {
	out := map[int64]string{}
	if len(userIDs) == 0 {
		return out
	}
	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	target := "https://thumbnails.roblox.com/v1/users/avatar-headshot?userIds=" +
		strings.Join(ids, ",") + "&size=150x150&format=Png&isCircular=false"
	res, err := fetchWithRetry(ctx, c.http, newGetRequest(target), 3)
	if err != nil || res.Status != http.StatusOK {
		return out
	}
	var parsed struct {
		Data []struct {
			TargetID int64  `json:"targetId"`
			ImageURL string `json:"imageUrl"`
			State    string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return out
	}
	for _, d := range parsed.Data {
		if d.State == "Completed" && d.ImageURL != "" {
			out[d.TargetID] = d.ImageURL
		}
	}
	return out
}

type PresenceType int

const (
	PresenceOffline  PresenceType = 0
	PresenceOnline   PresenceType = 1
	PresenceInGame   PresenceType = 2
	PresenceInStudio PresenceType = 3
)

func (p PresenceType) String() string {
	switch p {
	case PresenceOffline:
		return "Offline"
	case PresenceOnline:
		return "Online"
	case PresenceInGame:
		return "In Game"
	case PresenceInStudio:
		return "In Studio"
	default:
		return "Unknown"
	}
}

type Presence struct {
	UserID       int64        `json:"userId"`
	Type         PresenceType `json:"userPresenceType"`
	LastLocation string       `json:"lastLocation"`
	PlaceID      int64        `json:"placeId"`
	RootPlaceID  int64        `json:"rootPlaceId"`
	UniverseID   int64        `json:"universeId"`
	GameID       string       `json:"gameId"`
	LastOnline   time.Time    `json:"lastOnline"`
}

func (c *RobloxClient) GetPresences(ctx context.Context, userIDs []int64) (map[int64]Presence, error) {
	out := map[int64]Presence{}
	if len(userIDs) == 0 {
		return out, nil
	}
	target := "https://presence.roblox.com/v1/presence/users"
	res, err := fetchWithRetry(ctx, c.http, newJSONPost(target, map[string]any{"userIds": userIDs}), 5)
	if err != nil {
		return out, err
	}
	if res.Status != http.StatusOK {
		return out, fmt.Errorf("presence API returned status %d", res.Status)
	}
	var parsed struct {
		UserPresences []Presence `json:"userPresences"`
	}
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		return out, fmt.Errorf("parsing presence response: %w", err)
	}
	for _, p := range parsed.UserPresences {
		out[p.UserID] = p
	}
	return out, nil
}
