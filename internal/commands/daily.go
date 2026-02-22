package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cedaesca/alicia/internal/discord"
)

type dailyCommand struct {
	configStore NotificationConfigStore
}

func NewDailyCommand(configStore NotificationConfigStore) Command {
	return &dailyCommand{configStore: configStore}
}

func (command *dailyCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "daily",
		Description: "Crea una notificación diaria",
		Options: []discord.SlashCommandOption{
			{
				Name:        "base_hour",
				Description: "Hora base en UTC, formato HH:MM (24h)",
				Type:        discord.SlashCommandOptionTypeString,
				Required:    true,
			},
			{
				Name:        "title",
				Description: "Título de la notificación",
				Type:        discord.SlashCommandOptionTypeString,
				Required:    true,
			},
			{
				Name:        "message",
				Description: "Mensaje de la notificación",
				Type:        discord.SlashCommandOptionTypeString,
				Required:    true,
			},
		},
	}
}

func (command *dailyCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	baseHour := interaction.Options["base_hour"]
	if baseHour == "" {
		return "", MissingRequiredOptionError("base_hour")
	}

	title := strings.TrimSpace(interaction.Options["title"])
	if title == "" {
		return "", MissingRequiredOptionError("title")
	}

	message := strings.TrimSpace(interaction.Options["message"])
	if message == "" {
		return "", MissingRequiredOptionError("message")
	}

	if _, err := time.Parse("15:04", baseHour); err != nil {
		return "", fmt.Errorf("el valor base_hour debe tener formato HH:MM (24h) en UTC")
	}

	id, err := command.configStore.AddDailyNotification(ctx, interaction.GuildID, DailyNotificationInput{
		BaseHour: baseHour,
		Title:    title,
		Message:  message,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Notificación creada correctamente (hora base en UTC). ID: %s", id), nil
}
