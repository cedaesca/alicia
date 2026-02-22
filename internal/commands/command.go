package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/cedaesca/alicia/internal/discord"
)

var ErrCommandOnlyInGuild = errors.New("el comando solo puede usarse dentro del servidor")

func MissingRequiredOptionError(optionName string) error {
	return fmt.Errorf("falta opción obligatoria: %s", optionName)
}

type Command interface {
	Definition() discord.SlashCommand
	Execute(ctx context.Context, interaction discord.Interaction) (string, error)
}

type MessageSender interface {
	SendMessage(channelID, content string) error
}

func All(configStore NotificationConfigStore, messageSender MessageSender) []Command {
	return []Command{
		NewPingCommand(),
		NewSetChannelCommand(configStore, messageSender),
		NewNotificationRoleCommand(configStore),
		NewByMinutesCommand(configStore),
		NewDailyCommand(configStore),
		NewListCommand(configStore),
		NewDeleteCommand(configStore),
	}
}
