package commands

import (
	"context"
	"fmt"

	"github.com/cedaesca/alicia/internal/discord"
)

type setChannelCommand struct {
	configStore   NotificationConfigStore
	messageSender MessageSender
}

func NewSetChannelCommand(configStore NotificationConfigStore, messageSender MessageSender) Command {
	return &setChannelCommand{configStore: configStore, messageSender: messageSender}
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

	if command.messageSender != nil {
		if err := command.messageSender.SendMessage(channelID, "✅ Canal de notificaciones verificado."); err != nil {
			return "", fmt.Errorf("no tengo acceso al canal seleccionado; verifica permisos y que el bot esté en el servidor")
		}
	}

	if err := command.configStore.SetChannel(ctx, interaction.GuildID, channelID); err != nil {
		return "", err
	}

	return fmt.Sprintf("El canal de notificación ha sido establecido en <#%s>", channelID), nil
}
