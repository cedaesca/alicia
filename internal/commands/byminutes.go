package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cedaesca/alicia/internal/discord"
)

type byMinutesCommand struct {
	configStore NotificationConfigStore
}

func NewByMinutesCommand(configStore NotificationConfigStore) Command {
	return &byMinutesCommand{configStore: configStore}
}

func (command *byMinutesCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "byminutes",
		Description: "Crea una notificación recurrente por minutos",
		Options: []discord.SlashCommandOption{
			{
				Name:        "every_minutes",
				Description: "Cada cuántos minutos se enviará la notificación",
				Type:        discord.SlashCommandOptionTypeInteger,
				Required:    true,
			},
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

func (command *byMinutesCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	everyMinutesRaw := interaction.Options["every_minutes"]
	if everyMinutesRaw == "" {
		return "", MissingRequiredOptionError("every_minutes")
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

	everyMinutes, err := strconv.Atoi(everyMinutesRaw)
	if err != nil || everyMinutes <= 0 {
		return "", fmt.Errorf("el valor every_minutes debe ser un número entero mayor a 0")
	}

	if _, err := time.Parse("15:04", baseHour); err != nil {
		return "", fmt.Errorf("el valor base_hour debe tener formato HH:MM (24h) en UTC")
	}

	id, err := command.configStore.AddByMinutesNotification(ctx, interaction.GuildID, ByMinutesNotificationInput{
		EveryMinutes: everyMinutes,
		BaseHour:     baseHour,
		Title:        title,
		Message:      message,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Notificación creada correctamente (hora base en UTC). ID: %s", id), nil
}
