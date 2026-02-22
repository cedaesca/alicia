package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cedaesca/alicia/internal/discord"
)

type fakeNotificationConfigStore struct {
	setChannelErr   error
	setRoleErr      error
	addByMinutesErr error

	guildIDForChannel string
	channelID         string
	guildIDForRole    string
	roleID            string
	byMinutesGuildID  string
	byMinutesInput    ByMinutesNotificationInput
	byMinutesID       string
}

type fakeMessageSender struct {
	sendErr       error
	sentChannelID string
	sentContent   string
}

func (sender *fakeMessageSender) SendMessage(channelID, content string) error {
	sender.sentChannelID = channelID
	sender.sentContent = content
	return sender.sendErr
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

func (store *fakeNotificationConfigStore) AddByMinutesNotification(_ context.Context, guildID string, input ByMinutesNotificationInput) (string, error) {
	store.byMinutesGuildID = guildID
	store.byMinutesInput = input
	if store.addByMinutesErr != nil {
		return "", store.addByMinutesErr
	}

	if store.byMinutesID == "" {
		return "abc123", nil
	}

	return store.byMinutesID, nil
}

func (store *fakeNotificationConfigStore) GetGuildConfig(_ context.Context, guildID string) (NotificationConfig, error) {
	return NotificationConfig{}, nil
}

func (store *fakeNotificationConfigStore) ListDueNotifications(_ context.Context, now time.Time) ([]ScheduledNotification, error) {
	return nil, nil
}

func (store *fakeNotificationConfigStore) MarkNotificationSent(_ context.Context, notificationID string, sentAt time.Time) error {
	return nil
}

func TestSetChannelCommandExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := &fakeNotificationConfigStore{}
		sender := &fakeMessageSender{}
		command := NewSetChannelCommand(store, sender)

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

		if sender.sentChannelID != "channel-1" {
			t.Fatalf("expected verification message to channel-1, got %q", sender.sentChannelID)
		}
	})

	t.Run("fails outside guild", func(t *testing.T) {
		command := NewSetChannelCommand(&fakeNotificationConfigStore{}, &fakeMessageSender{})

		_, err := command.Execute(context.Background(), discord.Interaction{Options: map[string]string{"channel": "channel-1"}})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when bot has no access to selected channel", func(t *testing.T) {
		store := &fakeNotificationConfigStore{}
		sender := &fakeMessageSender{sendErr: errors.New("HTTP 403 Forbidden")}
		command := NewSetChannelCommand(store, sender)

		_, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{"channel": "channel-1"},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if store.channelID != "" {
			t.Fatalf("expected channel not to be persisted, got %q", store.channelID)
		}
	})

	t.Run("fails when store returns error", func(t *testing.T) {
		expectedErr := errors.New("write failed")
		command := NewSetChannelCommand(&fakeNotificationConfigStore{setChannelErr: expectedErr}, &fakeMessageSender{})

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

		if response != "Rol de notificación configurado a <@&role-1>" {
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
	all := All(&fakeNotificationConfigStore{}, nil)
	if len(all) < 4 {
		t.Fatalf("expected at least 4 commands, got %d", len(all))
	}
}

func TestByMinutesCommandExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := &fakeNotificationConfigStore{byMinutesID: "n1a2b3"}
		command := NewByMinutesCommand(store)

		response, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{
				"every_minutes": "240",
				"base_hour":     "16:00",
				"title":         "Recordatorio",
				"message":       "Enviar reporte",
			},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if response != "Notificación creada correctamente (hora base en UTC). ID: n1a2b3" {
			t.Fatalf("unexpected response: %q", response)
		}

		if store.byMinutesGuildID != "guild-1" {
			t.Fatalf("unexpected guild id: %q", store.byMinutesGuildID)
		}

		if store.byMinutesInput.EveryMinutes != 240 || store.byMinutesInput.BaseHour != "16:00" || store.byMinutesInput.Title != "Recordatorio" || store.byMinutesInput.Message != "Enviar reporte" {
			t.Fatalf("unexpected byminutes payload: %+v", store.byMinutesInput)
		}
	})

	t.Run("fails with invalid minutes", func(t *testing.T) {
		command := NewByMinutesCommand(&fakeNotificationConfigStore{})

		_, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{
				"every_minutes": "0",
				"base_hour":     "16:00",
				"title":         "Recordatorio",
				"message":       "Enviar reporte",
			},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails with invalid base hour", func(t *testing.T) {
		command := NewByMinutesCommand(&fakeNotificationConfigStore{})

		_, err := command.Execute(context.Background(), discord.Interaction{
			GuildID: "guild-1",
			Options: map[string]string{
				"every_minutes": "240",
				"base_hour":     "99:00",
				"title":         "Recordatorio",
				"message":       "Enviar reporte",
			},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
