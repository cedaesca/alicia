package commands

import (
	"context"
	"fmt"

	"github.com/cedaesca/alicia/internal/discord"
)

type notificationRoleCommand struct {
	configStore NotificationConfigStore
}

func NewNotificationRoleCommand(configStore NotificationConfigStore) Command {
	return &notificationRoleCommand{configStore: configStore}
}

func (command *notificationRoleCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "notificationrole",
		Description: "Configura el rol que se tageará para la notificación",
		Options: []discord.SlashCommandOption{
			{
				Name:        "role",
				Description: "Rol a mencionar en la notificación",
				Type:        discord.SlashCommandOptionTypeRole,
				Required:    true,
			},
		},
	}
}

func (command *notificationRoleCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	roleID := interaction.Options["role"]
	if roleID == "" {
		return "", MissingRequiredOptionError("role")
	}

	if err := command.configStore.SetRole(ctx, interaction.GuildID, roleID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Rol de notificación configurado a <@&%s>", roleID), nil
}
