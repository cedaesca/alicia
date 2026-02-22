package scheduler

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cedaesca/alicia/internal/commands"
	"github.com/cedaesca/alicia/internal/discord"
)

type fakeDiscordClient struct {
	sentChannelID string
	sentContent   string
	sendCalls     int
}

func (client *fakeDiscordClient) Open() error { return nil }

func (client *fakeDiscordClient) Close() error { return nil }

func (client *fakeDiscordClient) AddMessageCreateHandler(handler discord.MessageCreateHandler) {}

func (client *fakeDiscordClient) AddInteractionCreateHandler(handler discord.InteractionCreateHandler) {
}

func (client *fakeDiscordClient) ListGlobalCommands() ([]discord.RegisteredSlashCommand, error) {
	return nil, nil
}

func (client *fakeDiscordClient) RegisterGlobalCommand(command discord.SlashCommand) (string, error) {
	return "", nil
}

func (client *fakeDiscordClient) RespondToInteraction(interaction discord.Interaction, content string) error {
	return nil
}

func (client *fakeDiscordClient) SendMessage(channelID, content string) error {
	client.sentChannelID = channelID
	client.sentContent = content
	client.sendCalls++
	return nil
}

type fakeNotificationStore struct {
	dueNotifications []commands.ScheduledNotification
	guildConfig      commands.NotificationConfig
	markedID         string
	markedSentAt     time.Time
	markCalls        int
}

func (store *fakeNotificationStore) SetChannel(_ context.Context, guildID, channelID string) error {
	return nil
}

func (store *fakeNotificationStore) SetRole(_ context.Context, guildID, roleID string) error {
	return nil
}

func (store *fakeNotificationStore) AddByMinutesNotification(_ context.Context, guildID string, input commands.ByMinutesNotificationInput) (string, error) {
	return "", nil
}

func (store *fakeNotificationStore) AddDailyNotification(_ context.Context, guildID string, input commands.DailyNotificationInput) (string, error) {
	return "", nil
}

func (store *fakeNotificationStore) GetGuildConfig(_ context.Context, guildID string) (commands.NotificationConfig, error) {
	return store.guildConfig, nil
}

func (store *fakeNotificationStore) ListDueNotifications(_ context.Context, now time.Time) ([]commands.ScheduledNotification, error) {
	return store.dueNotifications, nil
}

func (store *fakeNotificationStore) MarkNotificationSent(_ context.Context, notificationID string, sentAt time.Time) error {
	store.markedID = notificationID
	store.markedSentAt = sentAt
	store.markCalls++
	return nil
}

func (store *fakeNotificationStore) ListGuildNotifications(_ context.Context, guildID string) ([]commands.ScheduledNotification, error) {
	return nil, nil
}

func (store *fakeNotificationStore) DeleteNotification(_ context.Context, guildID, notificationID string) error {
	return nil
}

func TestProcessDueNotificationsStartupSkipsStale(t *testing.T) {
	store := &fakeNotificationStore{
		dueNotifications: []commands.ScheduledNotification{
			{
				ID:                 "n1",
				GuildID:            "g1",
				Type:               "byminutes",
				EveryMinutes:       60,
				Message:            "hello",
				NextNotificationAt: time.Now().UTC().Add(-11 * time.Minute),
			},
		},
		guildConfig: commands.NotificationConfig{ChannelID: "c1"},
	}
	client := &fakeDiscordClient{}
	service := &NotificationService{
		ctx:           context.Background(),
		logger:        log.New(io.Discard, "", 0),
		discordClient: client,
		store:         store,
	}

	service.processDueNotifications(true)

	if client.sendCalls != 0 {
		t.Fatalf("expected no messages sent, got %d", client.sendCalls)
	}

	if store.markCalls != 1 {
		t.Fatalf("expected one mark call, got %d", store.markCalls)
	}

	if store.markedID != "n1" {
		t.Fatalf("expected marked id n1, got %q", store.markedID)
	}
}

func TestProcessDueNotificationsStartupSendsRecent(t *testing.T) {
	store := &fakeNotificationStore{
		dueNotifications: []commands.ScheduledNotification{
			{
				ID:                 "n2",
				GuildID:            "g2",
				Type:               "daily",
				Message:            "ping",
				NextNotificationAt: time.Now().UTC().Add(-2 * time.Minute),
			},
		},
		guildConfig: commands.NotificationConfig{ChannelID: "c2"},
	}
	client := &fakeDiscordClient{}
	service := &NotificationService{
		ctx:           context.Background(),
		logger:        log.New(io.Discard, "", 0),
		discordClient: client,
		store:         store,
	}

	service.processDueNotifications(true)

	if client.sendCalls != 1 {
		t.Fatalf("expected one message sent, got %d", client.sendCalls)
	}

	if store.markCalls != 1 {
		t.Fatalf("expected one mark call, got %d", store.markCalls)
	}

	if store.markedID != "n2" {
		t.Fatalf("expected marked id n2, got %q", store.markedID)
	}
}

func TestProcessDueNotificationsTickSendsStale(t *testing.T) {
	store := &fakeNotificationStore{
		dueNotifications: []commands.ScheduledNotification{
			{
				ID:                 "n3",
				GuildID:            "g3",
				Type:               "daily",
				Message:            "tick",
				NextNotificationAt: time.Now().UTC().Add(-30 * time.Minute),
			},
		},
		guildConfig: commands.NotificationConfig{ChannelID: "c3"},
	}
	client := &fakeDiscordClient{}
	service := &NotificationService{
		ctx:           context.Background(),
		logger:        log.New(io.Discard, "", 0),
		discordClient: client,
		store:         store,
	}

	service.processDueNotifications(false)

	if client.sendCalls != 1 {
		t.Fatalf("expected one message sent, got %d", client.sendCalls)
	}

	if store.markCalls != 1 {
		t.Fatalf("expected one mark call, got %d", store.markCalls)
	}

	if store.markedID != "n3" {
		t.Fatalf("expected marked id n3, got %q", store.markedID)
	}
}
