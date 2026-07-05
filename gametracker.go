package main

import (
	"context"
	"log"
	"time"
)

type GameTracker struct {
	cfg     *Config
	roblox  *RobloxClient
	discord *Discord
	state   *State

	universes map[string]string
	lastPing  map[string]time.Time
}

func newGameTracker(cfg *Config, roblox *RobloxClient, discord *Discord, state *State) *GameTracker {
	return &GameTracker{cfg: cfg, roblox: roblox, discord: discord, state: state, universes: map[string]string{}, lastPing: map[string]time.Time{}}
}

func (t *GameTracker) resolve(ctx context.Context) {
	for _, placeID := range t.cfg.Places {
		universeID, err := t.roblox.ResolveUniverseID(ctx, placeID)
		if err != nil {
			log.Printf("[games] could not resolve place %s: %v", placeID, err)
			continue
		}
		t.universes[universeID] = placeID
		log.Printf("[games] tracking place %s -> universe %s", placeID, universeID)
	}
}

func (t *GameTracker) universeIDs() []string {
	ids := make([]string, 0, len(t.universes))
	for u := range t.universes {
		ids = append(ids, u)
	}
	return ids
}

func (t *GameTracker) pollOnce(ctx context.Context) {
	ids := t.universeIDs()
	if len(ids) == 0 {
		return
	}

	games, err := t.roblox.GetGamesInfo(ctx, ids)
	if err != nil {
		log.Printf("[games] poll failed: %v", err)
		return
	}
	if len(games) == 0 {
		log.Printf("[games] poll returned no data")
		return
	}

	var changedIcons []string
	type changeEvent struct {
		game   GameInfo
		change *gameChange
	}
	changes := map[string]changeEvent{}

	for universeID, placeID := range t.universes {
		game, ok := games[universeID]
		if !ok {
			log.Printf("[games] no data returned for universe %s (place %s)", universeID, placeID)
			continue
		}

		prev, known := t.state.getGame(universeID)
		next := GameState{
			UniverseID:  universeID,
			PlaceID:     placeID,
			Name:        game.Name,
			Description: game.Description,
			Updated:     game.Updated,
			Seen:        true,
		}

		if t.cfg.Verbose {
			log.Printf("[games][verbose] universe=%s place=%s raw_fetch name=%q playing=%d visits=%d favorites=%d updated=%s desc_len=%d",
				universeID, placeID, game.Name, game.Playing, game.Visits, game.FavoritedCount,
				game.Updated.Format(time.RFC3339Nano), len(game.Description))
			log.Printf("[games][verbose] universe=%s known_in_state=%v prev_seen=%v", universeID, known, prev.Seen)
		}

		if !known || !prev.Seen {
			if t.cfg.Verbose {
				log.Printf("[games][verbose] universe=%s bootstrapping baseline, no comparison made this poll", universeID)
			}
			t.state.setGame(universeID, next)
			continue
		}

		if c := classifyGameChange(universeID, prev, next, t.cfg.Verbose); c != nil {
			changes[universeID] = changeEvent{game, c}
			changedIcons = append(changedIcons, universeID)
		}

		t.state.setGame(universeID, next)
	}

	if len(changes) == 0 {
		return
	}

	icons := t.roblox.GetGameIcons(ctx, changedIcons)
	for universeID, ev := range changes {
		placeID := t.universes[universeID]
		log.Printf("[games] change detected: %s (universe %s): %s", ev.game.Name, universeID, ev.change.Status)

		role := t.cfg.GameRole
		if role != "" {
			if last, ok := t.lastPing[universeID]; ok && time.Since(last) < t.cfg.GamePingCooldown {
				log.Printf("[games] ping suppressed for %s (universe %s): cooldown active (%s remaining)",
					ev.game.Name, universeID, (t.cfg.GamePingCooldown - time.Since(last)).Round(time.Second))
				role = ""
			} else {
				t.lastPing[universeID] = time.Now()
			}
		}

		e := gameEmbed(ev.game, placeID, icons[universeID], ev.change)
		if err := t.discord.send(ctx, t.cfg.GameWebhook, role, e); err != nil {
			log.Printf("[games] webhook send failed for %s: %v", ev.game.Name, err)
		}
	}

	if err := t.state.save(); err != nil {
		log.Printf("[games] failed to persist state: %v", err)
	}
}

// gameChange describes what differs between two consecutive snapshots of a
// tracked game, including the before/after values so the embed can show a
// detailed diff of exactly what was seen to change.
type gameChange struct {
	Status string

	NameChanged bool
	OldName     string
	NewName     string

	DescChanged bool
	OldDesc     string
	NewDesc     string

	UpdatedChanged bool
	OldUpdated     time.Time
	NewUpdated     time.Time
}

func classifyGameChange(universeID string, prev, next GameState, verbose bool) *gameChange {
	nameChanged := prev.Name != next.Name
	descChanged := prev.Description != next.Description
	updatedChanged := next.Updated.After(prev.Updated)

	if verbose {
		log.Printf("[games][verbose] universe=%s check=name prev=%q next=%q changed=%v", universeID, prev.Name, next.Name, nameChanged)
		log.Printf("[games][verbose] universe=%s check=description prev_len=%d next_len=%d changed=%v", universeID, len(prev.Description), len(next.Description), descChanged)
		if descChanged {
			log.Printf("[games][verbose] universe=%s description_prev=%q", universeID, prev.Description)
			log.Printf("[games][verbose] universe=%s description_next=%q", universeID, next.Description)
		}
		log.Printf("[games][verbose] universe=%s check=updated prev=%s next=%s prev_unix_nano=%d next_unix_nano=%d delta=%s changed=%v",
			universeID, prev.Updated.Format(time.RFC3339Nano), next.Updated.Format(time.RFC3339Nano),
			prev.Updated.UnixNano(), next.Updated.UnixNano(), next.Updated.Sub(prev.Updated), updatedChanged)
	}

	if !nameChanged && !descChanged && !updatedChanged {
		if verbose {
			log.Printf("[games][verbose] universe=%s result=no_change_detected", universeID)
		}
		return nil
	}

	c := &gameChange{
		NameChanged:    nameChanged,
		OldName:        prev.Name,
		NewName:        next.Name,
		DescChanged:    descChanged,
		OldDesc:        prev.Description,
		NewDesc:        next.Description,
		UpdatedChanged: updatedChanged,
		OldUpdated:     prev.Updated,
		NewUpdated:     next.Updated,
	}

	switch {
	case nameChanged && descChanged:
		c.Status = "Name and description were updated."
	case nameChanged:
		c.Status = "Game was renamed."
	case descChanged:
		c.Status = "Description was updated."
	default:
		c.Status = "Game's update timestamp changed, though the name and description are unchanged."
	}
	if verbose {
		log.Printf("[games][verbose] universe=%s result=change_detected status=%q", universeID, c.Status)
	}
	return c
}

func (t *GameTracker) Run(ctx context.Context) {
	t.resolve(ctx)
	if len(t.universes) == 0 {
		log.Printf("[games] no places resolved successfully; game tracking disabled")
		return
	}
	log.Printf("[games] tracking %d game(s), polling every %s", len(t.universes), t.cfg.PollInterval)

	t.pollOnce(ctx)
	if err := t.state.save(); err != nil {
		log.Printf("[games] failed to persist initial state: %v", err)
	}

	for {
		if err := sleepCtx(ctx, t.cfg.PollInterval); err != nil {
			return
		}
		t.pollOnce(ctx)
	}
}
