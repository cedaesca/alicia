package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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
	now := time.Now().UTC()
	for _, notification := range notifications {
		frequency := formatFrequency(notification)
		timeUntil := formatTimeUntilNotification(notification.NextNotificationAt, now)
		lines = append(lines, fmt.Sprintf("- **(%s) - %s** | Próxima en: %s | Frecuencia: %s", notification.ID, notification.Title, timeUntil, frequency))
	}

	return strings.Join(lines, "\n"), nil
}

func formatFrequency(notification ScheduledNotification) string {
	if notification.Type == "daily" {
		return "diaria"
	}

	return fmt.Sprintf("cada %d min", notification.EveryMinutes)
}

func formatTimeUntilNotification(nextNotificationAt time.Time, now time.Time) string {
	duration := nextNotificationAt.UTC().Sub(now)
	if duration < 0 {
		duration = 0
	}

	totalSeconds := int64(duration.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%d horas, %d minutos y %d segundos", hours, minutes, seconds)
}
