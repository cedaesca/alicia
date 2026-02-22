package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONNotificationConfigStore(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "notification_config.json")
	store := NewJSONNotificationConfigStore(filePath)

	if err := store.SetChannel(context.Background(), "guild-1", "channel-1"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if err := store.SetRole(context.Background(), "guild-1", "role-1"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("expected config file to exist, got %v", err)
	}

	var state notificationConfigState
	if err := json.Unmarshal(content, &state); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}

	guildConfig := state.Guilds["guild-1"]
	if guildConfig.ChannelID != "channel-1" {
		t.Fatalf("expected channel-1, got %q", guildConfig.ChannelID)
	}

	if guildConfig.RoleID != "role-1" {
		t.Fatalf("expected role-1, got %q", guildConfig.RoleID)
	}
}

func TestJSONNotificationConfigStoreHandlesInvalidJSON(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "notification_config.json")
	if err := os.WriteFile(filePath, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("failed to prepare invalid json file: %v", err)
	}

	store := NewJSONNotificationConfigStore(filePath)

	err := store.SetChannel(context.Background(), "guild-1", "channel-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
