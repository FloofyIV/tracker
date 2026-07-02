package main

import (
	"context"
	"log"
)

type GameTracker struct {
	cfg     *Config
	roblox  *RobloxClient
	discord *Discord
	state   *State

	universes map[string]string
}

func newGameTracker(cfg *Config, roblox *RobloxClient, discord *Discord, state *State) *GameTracker {
	return &GameTracker{cfg: cfg, roblox: roblox, discord: discord, state: state, universes: map[string]string{}}
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
	changes := map[string]struct {
		game   GameInfo
		status string
		detail string
	}{}

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

		if !known || !prev.Seen {
			t.state.setGame(universeID, next)
			continue
		}

		if changed, status, detail := classifyGameChange(prev, next); changed {
			changes[universeID] = struct {
				game   GameInfo
				status string
				detail string
			}{game, status, detail}
			changedIcons = append(changedIcons, universeID)
		}

		t.state.setGame(universeID, next)
	}

	if len(changes) == 0 {
		return
	}

	icons := t.roblox.GetGameIcons(ctx, changedIcons)
	for universeID, c := range changes {
		placeID := t.universes[universeID]
		log.Printf("[games] change detected: %s (universe %s)", c.game.Name, universeID)
		e := gameEmbed(c.game, placeID, icons[universeID], c.status, c.detail)
		if err := t.discord.send(ctx, t.cfg.GameWebhook, t.cfg.GameRole, e); err != nil {
			log.Printf("[games] webhook send failed for %s: %v", c.game.Name, err)
		}
	}

	if err := t.state.save(); err != nil {
		log.Printf("[games] failed to persist state: %v", err)
	}
}

func classifyGameChange(prev, next GameState) (changed bool, status, detail string) {
	nameChanged := prev.Name != next.Name
	descChanged := prev.Description != next.Description
	updatedChanged := next.Updated.After(prev.Updated)

	if !nameChanged && !descChanged && !updatedChanged {
		return false, "", ""
	}

	status = "Game metadata was refreshed."
	switch {
	case nameChanged && descChanged:
		status = "Name and description were updated."
		detail = next.Description
	case nameChanged:
		status = "Renamed from **" + prev.Name + "**."
	case descChanged:
		status = "Description was updated."
		detail = next.Description
	}
	return true, status, detail
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
