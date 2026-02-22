package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cedaesca/alicia/internal/discord"
)

type listCommand struct {
	configStore NotificationConfigStore
}

func NewListCommand(configStore NotificationConfigStore) Command {
	return &listCommand{configStore: configStore}
}

func (command *listCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "list",
		Description: "Lista notificaciones activas",
	}
}

func (command *listCommand) Execute(ctx context.Context, interaction discord.Interaction) (string, error) {
	if interaction.GuildID == "" {
		return "", ErrCommandOnlyInGuild
	}

	notifications, err := command.configStore.ListGuildNotifications(ctx, interaction.GuildID)
	if err != nil {
		return "", err
	}

	if len(notifications) == 0 {
		return "No hay notificaciones configuradas.", nil
	}

	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].ID < notifications[j].ID
	})

	lines := make([]string, 0, len(notifications)+1)
	lines = append(lines, "Notificaciones:")
	for _, notification := range notifications {
		frequency := formatFrequency(notification)
		notificationType := formatNotificationType(notification)
		lines = append(lines, fmt.Sprintf("- ID: %s | Título: %s | Tipo: %s | Frecuencia: %s", notification.ID, notification.Title, notificationType, frequency))
	}

	return strings.Join(lines, "\n"), nil
}

func formatFrequency(notification ScheduledNotification) string {
	if notification.Type == "daily" {
		return "diaria"
	}

	return fmt.Sprintf("cada %d min", notification.EveryMinutes)
}

func formatNotificationType(notification ScheduledNotification) string {
	if notification.Type == "daily" {
		return "daily"
	}

	if notification.Type == "byminutes" {
		return "byminutes"
	}

	return "desconocido"
}
