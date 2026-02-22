package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/cedaesca/alicia/internal/discord"
)

type deleteCommand struct {
	configStore NotificationConfigStore
}

func NewDeleteCommand(configStore NotificationConfigStore) Command {
	return &deleteCommand{configStore: configStore}
}

func (command *deleteCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "delete",
		Description: "Elimina una notificación por ID",
		Options: []discord.SlashCommandOption{
			{
				Name:        "id",
				Description: "ID de la notificación",
				Type:        discord.SlashCommandOptionTypeString,
				Required:    true,
			},
		},
	}
}

func (command *deleteCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	notificationID := strings.TrimSpace(interaction.Options["id"])
	if notificationID == "" {
		return "", MissingRequiredOptionError("id")
	}

	if err := command.configStore.DeleteNotification(ctx, interaction.GuildID, notificationID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Notificación eliminada: %s", notificationID), nil
}
