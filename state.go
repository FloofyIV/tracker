package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type GameState struct {
	UniverseID  string    `json:"universeId"`
	PlaceID     string    `json:"placeId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Updated     time.Time `json:"updated"`
	Seen        bool      `json:"seen"`
}

type UserState struct {
	UserID   int64        `json:"userId"`
	Username string       `json:"username"`
	Presence PresenceType `json:"presence"`
	PlaceID  int64        `json:"placeId"`
	GameName string       `json:"gameName"`

	Following map[int64]string `json:"following,omitempty"`

	Seen bool `json:"seen"`
}

type State struct {
	mu    sync.Mutex
	path  string
	Games map[string]GameState `json:"games"`
	Users map[string]UserState `json:"users"`
}

func loadState(path string) (*State, error) {
	s := &State{
		path:  path,
		Games: map[string]GameState{},
		Users: map[string]UserState{},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Games == nil {
		s.Games = map[string]GameState{}
	}
	if s.Users == nil {
		s.Users = map[string]UserState{}
	}
	return s, nil
}

func (s *State) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, ".tracker-state-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func (s *State) getGame(universeID string) (GameState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.Games[universeID]
	return g, ok
}

func (s *State) setGame(universeID string, g GameState) {
	s.mu.Lock()
	s.Games[universeID] = g
	s.mu.Unlock()
}

func (s *State) getUser(userID string) (UserState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.Users[userID]
	return u, ok
}

func (s *State) setUser(userID string, u UserState) {
	s.mu.Lock()
	s.Users[userID] = u
	s.mu.Unlock()
}
