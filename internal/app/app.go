package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedaesca/alicia/internal/commands"
	"github.com/cedaesca/alicia/internal/discord"
)

const commandStateFilePath = "data/discord_commands.json"
const notificationConfigFilePath = "data/notification_config.json"

type commandState struct {
	Commands map[string]string `json:"commands"`
}

type Application struct {
	ctx           context.Context
	logger        *log.Logger
	discordClient discord.Client
	commands      map[string]commands.Command
	stateFilePath string
}

func NewApplication(ctx context.Context, token string) (*Application, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(token) == "" {
		return nil, errors.New("missing bot token")
	}

	logger := log.New(os.Stdout, "[alicia] ", log.LstdFlags)

	discordClient, err := discord.NewDiscordGoClient(token)
	if err != nil {
		return nil, err
	}

	configStore := commands.NewJSONNotificationConfigStore(notificationConfigFilePath)

	registeredCommands := make(map[string]commands.Command)
	for _, command := range commands.All(configStore) {
		definition := command.Definition()
		registeredCommands[definition.Name] = command
	}

	return &Application{
		ctx:           ctx,
		logger:        logger,
		discordClient: discordClient,
		commands:      registeredCommands,
		stateFilePath: commandStateFilePath,
	}, nil
}

func (application *Application) Context() context.Context {
	return application.ctx
}

func (application *Application) Logger() *log.Logger {
	return application.logger
}

func (application *Application) DiscordClient() discord.Client {
	return application.discordClient
}

func (application *Application) Run() error {
	application.registerCommandHandler()

	application.logger.Println("Starting Discord client")

	if err := application.discordClient.Open(); err != nil {
		return err
	}

	if err := application.syncSlashCommands(); err != nil {
		_ = application.discordClient.Close()
		return fmt.Errorf("sync slash commands: %w", err)
	}

	application.logger.Println("Discord client is running")
	return nil
}

func (application *Application) Shutdown(ctx context.Context) error {
	application.logger.Println("Shutting down Discord client")

	errChan := make(chan error, 1)
	go func() {
		errChan <- application.discordClient.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			application.logger.Println("Discord client shutdown complete")
		}

		return err
	}
}

func (application *Application) registerCommandHandler() {
	application.discordClient.AddInteractionCreateHandler(func(interaction discord.Interaction) {
		command, ok := application.commands[interaction.CommandName]
		if !ok {
			application.logger.Printf("unknown command received: %s", interaction.CommandName)
			_ = application.discordClient.RespondToInteraction(interaction, "Unknown command")
			return
		}

		response, err := command.Execute(application.ctx, interaction)
		if err != nil {
			application.logger.Printf("failed to execute command %s: %v", interaction.CommandName, err)
			_ = application.discordClient.RespondToInteraction(interaction, "Something went wrong")
			return
		}

		if err := application.discordClient.RespondToInteraction(interaction, response); err != nil {
			application.logger.Printf("failed to respond to command %s: %v", interaction.CommandName, err)
		}
	})
}

func (application *Application) syncSlashCommands() error {
	state, err := application.loadCommandState()
	if err != nil {
		return err
	}

	loadedFromState := 0
	registeredNow := 0

	for name, command := range application.commands {
		if _, ok := state.Commands[name]; ok {
			loadedFromState++
			continue
		}

		commandID, err := application.discordClient.RegisterGlobalCommand(command.Definition())
		if err != nil {
			return err
		}

		state.Commands[name] = commandID
		registeredNow++
		application.logger.Printf("registered slash command: %s", name)
	}

	application.logger.Printf("slash commands ready: loaded=%d registered=%d", loadedFromState, registeredNow)

	return application.saveCommandState(state)
}

func (application *Application) loadCommandState() (commandState, error) {
	content, err := os.ReadFile(application.stateFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return commandState{Commands: make(map[string]string)}, nil
		}

		return commandState{}, err
	}

	var state commandState
	if err := json.Unmarshal(content, &state); err != nil {
		return commandState{}, err
	}

	if state.Commands == nil {
		state.Commands = make(map[string]string)
	}

	return state, nil
}

func (application *Application) saveCommandState(state commandState) error {
	if state.Commands == nil {
		state.Commands = make(map[string]string)
	}

	if err := os.MkdirAll(filepath.Dir(application.stateFilePath), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(application.stateFilePath, content, 0o644)
}
