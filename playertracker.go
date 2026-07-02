package main

import (
	"context"
	"log"
	"strconv"
	"sync"
)

type trackedUser struct {
	ID          int64
	Username    string
	DisplayName string
}

type PlayerTracker struct {
	cfg     *Config
	roblox  *RobloxClient
	discord *Discord
	state   *State

	users         []trackedUser
	universeCache map[int64]string
}

func newPlayerTracker(cfg *Config, roblox *RobloxClient, discord *Discord, state *State) *PlayerTracker {
	return &PlayerTracker{cfg: cfg, roblox: roblox, discord: discord, state: state, universeCache: map[int64]string{}}
}

func (t *PlayerTracker) resolve(ctx context.Context) {
	ids, err := t.roblox.ResolveUsernamesToIDs(ctx, t.cfg.Users)
	if err != nil {
		log.Printf("[players] username resolution failed: %v", err)
	}
	for _, handle := range t.cfg.Users {
		id, ok := ids[handle]
		if !ok {
			log.Printf("[players] could not resolve %q to a user ID", handle)
			continue
		}
		username := handle
		if _, err := strconv.ParseInt(handle, 10, 64); err == nil {
			username = "User " + handle
		}
		t.users = append(t.users, trackedUser{ID: id, Username: username})
		log.Printf("[players] tracking %s (userId %d)", username, id)
	}
}

func (t *PlayerTracker) userIDs() []int64 {
	ids := make([]int64, len(t.users))
	for i, u := range t.users {
		ids[i] = u.ID
	}
	return ids
}

func (t *PlayerTracker) gameNameFor(ctx context.Context, placeID int64) string {
	universeID, ok := t.universeCache[placeID]
	if !ok {
		var err error
		universeID, err = t.roblox.ResolveUniverseID(ctx, strconv.FormatInt(placeID, 10))
		if err != nil {
			return ""
		}
		t.universeCache[placeID] = universeID
	}
	games, err := t.roblox.GetGamesInfo(ctx, []string{universeID})
	if err != nil {
		return "a game"
	}
	if g, ok := games[universeID]; ok {
		return g.Name
	}
	return "a game"
}

func classifyPresenceChange(prevPresence, nextPresence PresenceType) string {
	switch {
	case prevPresence != PresenceInGame && nextPresence == PresenceInGame:
		return "join"
	case prevPresence == PresenceInGame && nextPresence != PresenceInGame:
		return "leave"
	case prevPresence == PresenceOffline && nextPresence != PresenceOffline && nextPresence != PresenceInGame:
		return "online"
	case prevPresence != PresenceOffline && nextPresence == PresenceOffline:
		return "offline"
	default:
		return ""
	}
}

func (t *PlayerTracker) pollPresenceOnce(ctx context.Context) {
	if len(t.users) == 0 {
		return
	}

	presences, err := t.roblox.GetPresences(ctx, t.userIDs())
	if err != nil {
		log.Printf("[players] presence poll failed: %v", err)
		return
	}

	type event struct {
		user     trackedUser
		kind     string
		gameName string
		placeID  int64
	}
	var events []event
	changed := false

	for _, u := range t.users {
		p, ok := presences[u.ID]
		presenceType := PresenceOffline
		placeID := int64(0)
		if ok {
			presenceType = p.Type
			placeID = p.RootPlaceID
			if placeID == 0 {
				placeID = p.PlaceID
			}
		}

		key := strconv.FormatInt(u.ID, 10)
		prev, known := t.state.getUser(key)

		next := prev
		next.UserID = u.ID
		next.Username = u.Username
		next.Presence = presenceType
		next.PlaceID = placeID
		next.GameName = ""
		next.Seen = true

		if !known || !prev.Seen {
			if presenceType == PresenceInGame && placeID != 0 {
				next.GameName = t.gameNameFor(ctx, placeID)
			}
			t.state.setUser(key, next)
			changed = true
			continue
		}

		switch classifyPresenceChange(prev.Presence, presenceType) {
		case "join":
			gameName := t.gameNameFor(ctx, placeID)
			next.GameName = gameName
			events = append(events, event{u, "join", gameName, placeID})

		case "leave":
			gameName := prev.GameName
			if gameName == "" {
				gameName = "a game"
			}
			events = append(events, event{u, "leave", gameName, prev.PlaceID})

		case "online":
			events = append(events, event{u, "online", "", 0})

		case "offline":
			events = append(events, event{u, "offline", "", 0})

		default:
			if presenceType == PresenceInGame {
				next.GameName = prev.GameName
			}
		}

		if prev.Presence != next.Presence || prev.PlaceID != next.PlaceID {
			changed = true
		}
		t.state.setUser(key, next)
	}

	if len(events) == 0 {
		if changed {
			if err := t.state.save(); err != nil {
				log.Printf("[players] failed to persist state: %v", err)
			}
		}
		return
	}

	avatars := t.roblox.GetAvatarHeadshots(ctx, t.userIDs())
	for _, ev := range events {
		log.Printf("[players] %s -> %s (%s)", ev.user.Username, ev.kind, ev.gameName)
		placeIDStr := ""
		if ev.placeID != 0 {
			placeIDStr = strconv.FormatInt(ev.placeID, 10)
		}
		e := playerEmbed(ev.user.Username, ev.user.ID, avatars[ev.user.ID], ev.kind, ev.gameName, placeIDStr)
		if err := t.discord.send(ctx, t.cfg.PlayerWebhook, t.cfg.PlayerRole, e); err != nil {
			log.Printf("[players] webhook send failed for %s: %v", ev.user.Username, err)
		}
	}

	if err := t.state.save(); err != nil {
		log.Printf("[players] failed to persist state: %v", err)
	}
}

func (t *PlayerTracker) Run(ctx context.Context) {
	t.resolve(ctx)
	if len(t.users) == 0 {
		log.Printf("[players] no users resolved successfully; player tracking disabled")
		return
	}
	log.Printf("[players] tracking %d player(s): presence every %s",
		len(t.users), t.cfg.PlayerInterval)

	t.pollPresenceOnce(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			if err := sleepCtx(ctx, t.cfg.PlayerInterval); err != nil {
				return
			}
			t.pollPresenceOnce(ctx)
		}
	}()

	wg.Wait()
}
