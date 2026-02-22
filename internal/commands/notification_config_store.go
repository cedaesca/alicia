package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type NotificationConfigStore interface {
	SetChannel(ctx context.Context, guildID, channelID string) error
	SetRole(ctx context.Context, guildID, roleID string) error
}

type NotificationConfig struct {
	ChannelID string `json:"channel_id,omitempty"`
	RoleID    string `json:"role_id,omitempty"`
}

type notificationConfigState struct {
	Guilds map[string]NotificationConfig `json:"guilds"`
}

type jsonNotificationConfigStore struct {
	filePath string
	mu       sync.Mutex
}

func NewJSONNotificationConfigStore(filePath string) NotificationConfigStore {
	return &jsonNotificationConfigStore{filePath: filePath}
}

func (store *jsonNotificationConfigStore) SetChannel(_ context.Context, guildID, channelID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.load()
	if err != nil {
		return err
	}

	config := state.Guilds[guildID]
	config.ChannelID = channelID
	state.Guilds[guildID] = config

	return store.save(state)
}

func (store *jsonNotificationConfigStore) SetRole(_ context.Context, guildID, roleID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.load()
	if err != nil {
		return err
	}

	config := state.Guilds[guildID]
	config.RoleID = roleID
	state.Guilds[guildID] = config

	return store.save(state)
}

func (store *jsonNotificationConfigStore) load() (notificationConfigState, error) {
	content, err := os.ReadFile(store.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return notificationConfigState{Guilds: make(map[string]NotificationConfig)}, nil
		}

		return notificationConfigState{}, err
	}

	var state notificationConfigState
	if err := json.Unmarshal(content, &state); err != nil {
		return notificationConfigState{}, err
	}

	if state.Guilds == nil {
		state.Guilds = make(map[string]NotificationConfig)
	}

	return state, nil
}

func (store *jsonNotificationConfigStore) save(state notificationConfigState) error {
	if state.Guilds == nil {
		state.Guilds = make(map[string]NotificationConfig)
	}

	if err := os.MkdirAll(filepath.Dir(store.filePath), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(store.filePath, content, 0o644)
}
