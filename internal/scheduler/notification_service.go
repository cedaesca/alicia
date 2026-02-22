package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cedaesca/alicia/internal/commands"
	"github.com/cedaesca/alicia/internal/discord"
)

const startupStaleNotificationThreshold = 10 * time.Minute

type NotificationService struct {
	ctx           context.Context
	logger        *log.Logger
	discordClient discord.Client
	store         commands.NotificationConfigStore
	interval      time.Duration
	cancel        context.CancelFunc
}

func NewNotificationService(ctx context.Context, logger *log.Logger, discordClient discord.Client, store commands.NotificationConfigStore) *NotificationService {
	if ctx == nil {
		ctx = context.Background()
	}

	return &NotificationService{
		ctx:           ctx,
		logger:        logger,
		discordClient: discordClient,
		store:         store,
		interval:      30 * time.Second,
	}
}

func (service *NotificationService) Start() {
	if service.store == nil {
		return
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	service.cancel = cancel

	go func() {
		ticker := time.NewTicker(service.interval)
		defer ticker.Stop()

		service.processDueNotifications(true)

		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				service.processDueNotifications(false)
			}
		}
	}()
}

func (service *NotificationService) Stop() {
	if service.cancel != nil {
		service.cancel()
	}
}

func (service *NotificationService) processDueNotifications(isStartup bool) {
	now := time.Now().UTC()
	dueNotifications, err := service.store.ListDueNotifications(service.ctx, now)
	if err != nil {
		service.logger.Printf("failed to list due notifications: %v", err)
		return
	}

	for _, notification := range dueNotifications {
		if isStartup && now.Sub(notification.NextNotificationAt.UTC()) > startupStaleNotificationThreshold {
			if err := service.store.MarkNotificationSent(service.ctx, notification.ID, now); err != nil {
				service.logger.Printf("failed to skip stale notification %s on startup: %v", notification.ID, err)
				continue
			}

			service.logger.Printf("notification skipped on startup (stale): id=%s guild=%s", notification.ID, notification.GuildID)
			continue
		}

		guildConfig, err := service.store.GetGuildConfig(service.ctx, notification.GuildID)
		if err != nil {
			service.logger.Printf("failed to load guild config for notification %s: %v", notification.ID, err)
			continue
		}

		if strings.TrimSpace(guildConfig.ChannelID) == "" {
			service.logger.Printf("notification %s skipped: no channel configured", notification.ID)
			continue
		}

		message := formatNotificationMessage(notification, guildConfig.RoleID)
		if err := service.discordClient.SendMessage(guildConfig.ChannelID, message); err != nil {
			service.logger.Printf("failed to send notification %s: %v", notification.ID, err)
			continue
		}

		if err := service.store.MarkNotificationSent(service.ctx, notification.ID, now); err != nil {
			service.logger.Printf("failed to update schedule for notification %s: %v", notification.ID, err)
			continue
		}

		service.logger.Printf("notification sent: id=%s guild=%s", notification.ID, notification.GuildID)
	}
}

func formatNotificationMessage(notification commands.ScheduledNotification, roleID string) string {
	prefix := ""
	if strings.TrimSpace(roleID) != "" {
		prefix = fmt.Sprintf("<@&%s> ", roleID)
	}

	return fmt.Sprintf("%s %s", prefix, notification.Message)
}
