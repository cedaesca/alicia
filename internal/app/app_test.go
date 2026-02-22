package app

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cedaesca/alicia/internal/commands"
	"github.com/cedaesca/alicia/internal/discord"
)

var nilContext context.Context

type fakeDiscordClient struct {
	openErr            error
	closeErr           error
	closeCh            chan struct{}
	interactionHandler discord.InteractionCreateHandler
	listCommandsErr    error
	registeredCommands []discord.SlashCommand
	existingCommands   []discord.RegisteredSlashCommand
}

func (client *fakeDiscordClient) Open() error {
	return client.openErr
}

func (client *fakeDiscordClient) Close() error {
	if client.closeCh != nil {
		<-client.closeCh
	}

	return client.closeErr
}

func (client *fakeDiscordClient) AddMessageCreateHandler(handler discord.MessageCreateHandler) {
}

func (client *fakeDiscordClient) AddInteractionCreateHandler(handler discord.InteractionCreateHandler) {
	client.interactionHandler = handler
}

func (client *fakeDiscordClient) ListGlobalCommands() ([]discord.RegisteredSlashCommand, error) {
	if client.listCommandsErr != nil {
		return nil, client.listCommandsErr
	}

	return client.existingCommands, nil
}

func (client *fakeDiscordClient) RegisterGlobalCommand(command discord.SlashCommand) (string, error) {
	client.registeredCommands = append(client.registeredCommands, command)
	return "command-id", nil
}

func (client *fakeDiscordClient) RespondToInteraction(interaction discord.Interaction, content string) error {
	return nil
}

type staticCommand struct {
	definition discord.SlashCommand
}

func (command *staticCommand) Definition() discord.SlashCommand {
	return command.definition
}

func (command *staticCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	return "ok", nil
}

func TestSyncSlashCommandsUsesLocalStateAsSourceOfTruth(t *testing.T) {
	tempDir := t.TempDir()

	application := &Application{
		ctx:           context.Background(),
		logger:        log.New(io.Discard, "", 0),
		discordClient: &fakeDiscordClient{},
		commands: map[string]commands.Command{
			"setchannel": &staticCommand{definition: discord.SlashCommand{Name: "setchannel", Description: "Set channel"}},
		},
		stateFilePath: filepath.Join(tempDir, "discord_commands.json"),
	}

	if err := os.MkdirAll(filepath.Dir(application.stateFilePath), 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	if err := os.WriteFile(application.stateFilePath, []byte(`{"commands":{"setchannel":"old-id"}}`), 0o644); err != nil {
		t.Fatalf("failed to seed state file: %v", err)
	}

	if err := application.syncSlashCommands(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	fakeClient := application.discordClient.(*fakeDiscordClient)
	if len(fakeClient.registeredCommands) != 0 {
		t.Fatalf("expected command not to be registered, got %d", len(fakeClient.registeredCommands))
	}
}

func (client *fakeDiscordClient) SendMessage(channelID, content string) error {
	return nil
}

func TestNewApplication(t *testing.T) {
	t.Run("uses background when context is nil", func(t *testing.T) {
		application, err := NewApplication(nilContext, "test-token")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if application.Context() == nil {
			t.Fatal("expected context to be initialized")
		}

		if application.Logger() == nil {
			t.Fatal("expected logger to be initialized")
		}

		if application.DiscordClient() == nil {
			t.Fatal("expected discord client to be initialized")
		}
	})

	t.Run("fails when token is missing", func(t *testing.T) {
		application, err := NewApplication(context.Background(), "   ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if application != nil {
			t.Fatal("expected nil application")
		}
	})
}

func TestApplicationRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{},
			commands:      map[string]commands.Command{},
			stateFilePath: filepath.Join(t.TempDir(), "commands.json"),
		}

		if err := application.Run(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error from discord client", func(t *testing.T) {
		expectedErr := errors.New("open failed")
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{openErr: expectedErr},
			commands:      map[string]commands.Command{},
			stateFilePath: filepath.Join(t.TempDir(), "commands.json"),
		}

		err := application.Run()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestApplicationShutdown(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{},
			commands:      map[string]commands.Command{},
			stateFilePath: filepath.Join(t.TempDir(), "commands.json"),
		}

		if err := application.Shutdown(context.Background()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error from discord client", func(t *testing.T) {
		expectedErr := errors.New("close failed")
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{closeErr: expectedErr},
			commands:      map[string]commands.Command{},
			stateFilePath: filepath.Join(t.TempDir(), "commands.json"),
		}

		err := application.Shutdown(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("context cancelled before close completes", func(t *testing.T) {
		closeCh := make(chan struct{})
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{closeCh: closeCh},
			commands:      map[string]commands.Command{},
			stateFilePath: filepath.Join(t.TempDir(), "commands.json"),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := application.Shutdown(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}

		close(closeCh)
	})
}

func TestResolveDataFilePath(t *testing.T) {
	t.Run("uses executable directory when provided", func(t *testing.T) {
		exePath := filepath.Join(t.TempDir(), "alicia.exe")
		path := resolveDataFilePath(exePath, "notifications.json")
		expected := filepath.Join(filepath.Dir(exePath), "data", "notifications.json")

		if path != expected {
			t.Fatalf("expected %q, got %q", expected, path)
		}
	})

	t.Run("falls back to relative data directory when executable path missing", func(t *testing.T) {
		path := resolveDataFilePath("", "notifications.json")
		expected := filepath.Join("data", "notifications.json")

		if path != expected {
			t.Fatalf("expected %q, got %q", expected, path)
		}
	})
}

func TestReadGuildAndNotificationCounts(t *testing.T) {
	t.Run("returns zero counts when files are missing", func(t *testing.T) {
		tempDir := t.TempDir()

		guildCount, notificationCount, err := readGuildAndNotificationCounts(
			filepath.Join(tempDir, "notification_config.json"),
			filepath.Join(tempDir, "notifications.json"),
		)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if guildCount != 0 || notificationCount != 0 {
			t.Fatalf("expected 0 guilds and 0 notifications, got %d and %d", guildCount, notificationCount)
		}
	})

	t.Run("returns counts from valid files", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "notification_config.json")
		notificationsPath := filepath.Join(tempDir, "notifications.json")

		if err := os.WriteFile(configPath, []byte(`{"guilds":{"g1":{},"g2":{}}}`), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		if err := os.WriteFile(notificationsPath, []byte(`{"notifications":[{"id":"a"},{"id":"b"},{"id":"c"}]}`), 0o644); err != nil {
			t.Fatalf("failed to write notifications file: %v", err)
		}

		guildCount, notificationCount, err := readGuildAndNotificationCounts(configPath, notificationsPath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if guildCount != 2 || notificationCount != 3 {
			t.Fatalf("expected 2 guilds and 3 notifications, got %d and %d", guildCount, notificationCount)
		}
	})

	t.Run("returns partial counts and error when one file is invalid", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "notification_config.json")
		notificationsPath := filepath.Join(tempDir, "notifications.json")

		if err := os.WriteFile(configPath, []byte(`{"guilds":{"g1":{}}}`), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		if err := os.WriteFile(notificationsPath, []byte(`not-json`), 0o644); err != nil {
			t.Fatalf("failed to write notifications file: %v", err)
		}

		guildCount, notificationCount, err := readGuildAndNotificationCounts(configPath, notificationsPath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if guildCount != 1 || notificationCount != 0 {
			t.Fatalf("expected 1 guild and 0 notifications, got %d and %d", guildCount, notificationCount)
		}
	})
}
