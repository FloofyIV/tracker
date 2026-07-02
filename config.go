package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Places       []string
	GameWebhook  string
	GameRole     string
	PollInterval time.Duration

	Users          []string
	PlayerWebhook  string
	PlayerRole     string
	PlayerInterval time.Duration

	StateFile  string
	HTTPTimout time.Duration
}

func env(def string, names ...string) string {
	for _, n := range names {
		if v := strings.TrimSpace(os.Getenv(n)); v != "" {
			return v
		}
	}
	return def
}

func envDuration(def time.Duration, name string) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}

func splitList(raw string) []string {
	if raw == "" {
		return nil
	}
	raw = strings.NewReplacer(";", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Places:         splitList(env("", "PLACES", "PLACE")),
		GameWebhook:    env("", "WEBHOOK", "GAME_WEBHOOK"),
		GameRole:       env("", "ROLE", "GAME_ROLE"),
		PollInterval:   envDuration(20*time.Second, "POLL_INTERVAL"),
		Users:          splitList(env("", "USERS", "PLAYERS")),
		PlayerWebhook:  env("", "PLAYER_WEBHOOK"),
		PlayerRole:     env("", "PLAYER_ROLE"),
		PlayerInterval: envDuration(30*time.Second, "PLAYER_POLL_INTERVAL"),
		StateFile:      env("state.json", "STATE_FILE"),
		HTTPTimout:     envDuration(15*time.Second, "HTTP_TIMEOUT"),
	}

	if cfg.PlayerWebhook == "" {
		cfg.PlayerWebhook = cfg.GameWebhook
	}
	if cfg.PlayerRole == "" {
		cfg.PlayerRole = cfg.GameRole
	}

	if len(cfg.Places) == 0 && len(cfg.Users) == 0 {
		return nil, fmt.Errorf("nothing to track, set PLACES and/or USERS")
	}
	if len(cfg.Places) > 0 && cfg.GameWebhook == "" {
		return nil, fmt.Errorf("PLACES is set but no WEBHOOK was provided")
	}
	if len(cfg.Users) > 0 && cfg.PlayerWebhook == "" {
		return nil, fmt.Errorf("USERS is set but no PLAYER_WEBHOOK/WEBHOOK was provided")
	}
	if cfg.PollInterval < 5*time.Second {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.PlayerInterval < 5*time.Second {
		cfg.PlayerInterval = 5 * time.Second
	}

	return cfg, nil
}
