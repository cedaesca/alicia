package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/cedaesca/alicia/internal/discord"
)

type fakeNotificationConfigStore struct {
	setChannelErr error
	setRoleErr    error

	guildIDForChannel string
	channelID         string
	guildIDForRole    string
	roleID            string
}

func (store *fakeNotificationConfigStore) SetChannel(_ context.Context, guildID, channelID string) error {
	store.guildIDForChannel = guildID
	store.channelID = channelID
	return store.setChannelErr
}

func (store *fakeNotificationConfigStore) SetRole(_ context.Context, guildID, roleID string) error {
	store.guildIDForRole = guildID
	store.roleID = roleID
	return store.setRoleErr
}

func TestSetChannelCommandExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := &fakeNotificationConfigStore{}
		command := NewSetChannelCommand(store)

		response, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{"channel": "channel-1"},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if response != "El canal de notificación ha sido establecido en <#channel-1>" {
			t.Fatalf("unexpected response: %q", response)
		}

		if store.guildIDForChannel != "guild-1" || store.channelID != "channel-1" {
			t.Fatalf("unexpected store payload: guild=%q channel=%q", store.guildIDForChannel, store.channelID)
		}
	})

	t.Run("fails outside guild", func(t *testing.T) {
		command := NewSetChannelCommand(&fakeNotificationConfigStore{})

		_, err := command.Execute(context.Background(), discord.Interaction{Options: map[string]string{"channel": "channel-1"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when store returns error", func(t *testing.T) {
		expectedErr := errors.New("write failed")
		command := NewSetChannelCommand(&fakeNotificationConfigStore{setChannelErr: expectedErr})

		_, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{"channel": "channel-1"},
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestNotificationRoleCommandExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := &fakeNotificationConfigStore{}
		command := NewNotificationRoleCommand(store)

		response, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{"role": "role-1"},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if response != "Notification role set to <@&role-1>" {
			t.Fatalf("unexpected response: %q", response)
		}

		if store.guildIDForRole != "guild-1" || store.roleID != "role-1" {
			t.Fatalf("unexpected store payload: guild=%q role=%q", store.guildIDForRole, store.roleID)
		}
	})

	t.Run("fails outside guild", func(t *testing.T) {
		command := NewNotificationRoleCommand(&fakeNotificationConfigStore{})

		_, err := command.Execute(context.Background(), discord.Interaction{Options: map[string]string{"role": "role-1"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when store returns error", func(t *testing.T) {
		expectedErr := errors.New("write failed")
		command := NewNotificationRoleCommand(&fakeNotificationConfigStore{setRoleErr: expectedErr})

		_, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{"role": "role-1"},
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestAllCommandsIncludesNotificationCommands(t *testing.T) {
	all := All(&fakeNotificationConfigStore{})
	if len(all) < 3 {
		t.Fatalf("expected at least 3 commands, got %d", len(all))
	}
}
