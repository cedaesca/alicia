package commands

import (
	"context"
	"fmt"

	"github.com/cedaesca/alicia/internal/discord"
)

type setChannelCommand struct {
	configStore NotificationConfigStore
}

func NewSetChannelCommand(configStore NotificationConfigStore) Command {
	return &setChannelCommand{configStore: configStore}
}

func (command *setChannelCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "setchannel",
		Description: "Configura el canal donde se enviarán las notificaciones",
		Options: []discord.SlashCommandOption{
			{
				Name:        "channel",
				Description: "El canal donde se enviarán las notificaciones",
				Type:        discord.SlashCommandOptionTypeChannel,
				Required:    true,
			},
		},
	}
}

func (command *setChannelCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	channelID := interaction.Options["channel"]
	if channelID == "" {
		return "", MissingRequiredOptionError("channel")
	}

	if err := command.configStore.SetChannel(ctx, interaction.GuildID, channelID); err != nil {
		return "", err
	}

	return fmt.Sprintf("El canal de notificación ha sido establecido en <#%s>", channelID), nil
}
