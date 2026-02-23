package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type NotificationConfigStore interface {
	SetChannel(ctx context.Context, guildID, channelID string) error
	SetRole(ctx context.Context, guildID, roleID string) error
	AddByMinutesNotification(ctx context.Context, guildID string, input ByMinutesNotificationInput) (string, error)
	AddDailyNotification(ctx context.Context, guildID string, input DailyNotificationInput) (string, error)
	GetGuildConfig(ctx context.Context, guildID string) (NotificationConfig, error)
	ListGuildNotifications(ctx context.Context, guildID string) ([]ScheduledNotification, error)
	DeleteNotification(ctx context.Context, guildID, notificationID string) error
	ListDueNotifications(ctx context.Context, now time.Time) ([]ScheduledNotification, error)
	MarkNotificationSent(ctx context.Context, notificationID string, sentAt time.Time) error
	RecalculateAllNextNotifications(ctx context.Context, now time.Time) error
}

type ByMinutesNotificationInput struct {
	EveryMinutes int
	BaseHour     string
	Title        string
	Message      string
}

type ByMinutesNotification struct {
	ID           string `json:"id"`
	EveryMinutes int    `json:"every_minutes"`
	BaseHour     string `json:"base_hour"`
	Title        string `json:"title"`
	Message      string `json:"message"`
}

type DailyNotificationInput struct {
	BaseHour string
	Title    string
	Message  string
}

type DailyNotification struct {
	ID       string `json:"id"`
	BaseHour string `json:"base_hour"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

type ScheduledNotification struct {
	ID                 string    `json:"id"`
	GuildID            string    `json:"guild_id"`
	Type               string    `json:"type"`
	EveryMinutes       int       `json:"every_minutes"`
	BaseHour           string    `json:"base_hour"`
	Title              string    `json:"title"`
	Message            string    `json:"message"`
	NextNotificationAt time.Time `json:"next_notification_at"`
}

type NotificationConfig struct {
	ChannelID              string                  `json:"channel_id,omitempty"`
	RoleID                 string                  `json:"role_id,omitempty"`
	ByMinutesNotifications []ByMinutesNotification `json:"by_minutes_notifications,omitempty"`
	DailyNotifications     []DailyNotification     `json:"daily_notifications,omitempty"`
}

type notificationConfigState struct {
	Guilds map[string]NotificationConfig `json:"guilds"`
}

type notificationScheduleState struct {
	Notifications []ScheduledNotification `json:"notifications"`
}

type jsonNotificationConfigStore struct {
	configFilePath        string
	notificationsFilePath string
	mu                    sync.Mutex
}

func NewJSONNotificationConfigStore(filePath string) NotificationConfigStore {
	return &jsonNotificationConfigStore{
		configFilePath:        filePath,
		notificationsFilePath: filepath.Join(filepath.Dir(filePath), "notifications.json"),
	}
}

func (store *jsonNotificationConfigStore) SetChannel(_ context.Context, guildID, channelID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadConfigState()
	if err != nil {
		return err
	}

	config := state.Guilds[guildID]
	config.ChannelID = channelID
	state.Guilds[guildID] = config

	return store.saveConfigState(state)
}

func (store *jsonNotificationConfigStore) SetRole(_ context.Context, guildID, roleID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadConfigState()
	if err != nil {
		return err
	}

	config := state.Guilds[guildID]
	config.RoleID = roleID
	state.Guilds[guildID] = config

	return store.saveConfigState(state)
}

func (store *jsonNotificationConfigStore) AddByMinutesNotification(_ context.Context, guildID string, input ByMinutesNotificationInput) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	configState, err := store.loadConfigState()
	if err != nil {
		return "", err
	}

	notificationState, err := store.loadNotificationScheduleState()
	if err != nil {
		return "", err
	}

	config := configState.Guilds[guildID]

	id, err := generateShortID()
	if err != nil {
		return "", err
	}

	config.ByMinutesNotifications = append(config.ByMinutesNotifications, ByMinutesNotification{
		ID:           id,
		EveryMinutes: input.EveryMinutes,
		BaseHour:     input.BaseHour,
		Title:        input.Title,
		Message:      input.Message,
	})

	nextNotificationAt, err := calculateInitialNextNotificationAt(input.BaseHour, input.EveryMinutes, time.Now().UTC())
	if err != nil {
		return "", err
	}

	notificationState.Notifications = append(notificationState.Notifications, ScheduledNotification{
		ID:                 id,
		GuildID:            guildID,
		Type:               "byminutes",
		EveryMinutes:       input.EveryMinutes,
		BaseHour:           input.BaseHour,
		Title:              input.Title,
		Message:            input.Message,
		NextNotificationAt: nextNotificationAt,
	})

	configState.Guilds[guildID] = config

	if err := store.saveConfigState(configState); err != nil {
		return "", err
	}

	if err := store.saveNotificationScheduleState(notificationState); err != nil {
		return "", err
	}

	return id, nil
}

func (store *jsonNotificationConfigStore) AddDailyNotification(_ context.Context, guildID string, input DailyNotificationInput) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	configState, err := store.loadConfigState()
	if err != nil {
		return "", err
	}

	notificationState, err := store.loadNotificationScheduleState()
	if err != nil {
		return "", err
	}

	config := configState.Guilds[guildID]

	id, err := generateShortID()
	if err != nil {
		return "", err
	}

	config.DailyNotifications = append(config.DailyNotifications, DailyNotification{
		ID:       id,
		BaseHour: input.BaseHour,
		Title:    input.Title,
		Message:  input.Message,
	})

	nextNotificationAt, err := calculateInitialDailyNextNotificationAt(input.BaseHour, time.Now().UTC())
	if err != nil {
		return "", err
	}

	notificationState.Notifications = append(notificationState.Notifications, ScheduledNotification{
		ID:                 id,
		GuildID:            guildID,
		Type:               "daily",
		EveryMinutes:       0,
		BaseHour:           input.BaseHour,
		Title:              input.Title,
		Message:            input.Message,
		NextNotificationAt: nextNotificationAt,
	})

	configState.Guilds[guildID] = config

	if err := store.saveConfigState(configState); err != nil {
		return "", err
	}

	if err := store.saveNotificationScheduleState(notificationState); err != nil {
		return "", err
	}

	return id, nil
}

func (store *jsonNotificationConfigStore) GetGuildConfig(_ context.Context, guildID string) (NotificationConfig, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadConfigState()
	if err != nil {
		return NotificationConfig{}, err
	}

	return state.Guilds[guildID], nil
}

func (store *jsonNotificationConfigStore) ListDueNotifications(_ context.Context, now time.Time) ([]ScheduledNotification, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadNotificationScheduleState()
	if err != nil {
		return nil, err
	}

	normalizedNow := now.UTC()
	dueNotifications := make([]ScheduledNotification, 0)
	for _, notification := range state.Notifications {
		if !notification.NextNotificationAt.After(normalizedNow) {
			dueNotifications = append(dueNotifications, notification)
		}
	}

	return dueNotifications, nil
}

func (store *jsonNotificationConfigStore) ListGuildNotifications(_ context.Context, guildID string) ([]ScheduledNotification, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadNotificationScheduleState()
	if err != nil {
		return nil, err
	}

	notifications := make([]ScheduledNotification, 0)
	for _, notification := range state.Notifications {
		if notification.GuildID == guildID {
			notifications = append(notifications, notification)
		}
	}

	return notifications, nil
}

func (store *jsonNotificationConfigStore) DeleteNotification(_ context.Context, guildID, notificationID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	configState, err := store.loadConfigState()
	if err != nil {
		return err
	}

	notificationState, err := store.loadNotificationScheduleState()
	if err != nil {
		return err
	}

	deleted := false
	filteredScheduled := make([]ScheduledNotification, 0, len(notificationState.Notifications))
	for _, notification := range notificationState.Notifications {
		if notification.ID == notificationID && notification.GuildID == guildID {
			deleted = true
			continue
		}

		filteredScheduled = append(filteredScheduled, notification)
	}

	if !deleted {
		return errors.New("notificación no encontrada")
	}

	notificationState.Notifications = filteredScheduled

	config := configState.Guilds[guildID]
	filteredConfig := make([]ByMinutesNotification, 0, len(config.ByMinutesNotifications))
	for _, notification := range config.ByMinutesNotifications {
		if notification.ID == notificationID {
			continue
		}

		filteredConfig = append(filteredConfig, notification)
	}

	config.ByMinutesNotifications = filteredConfig
	filteredDailyConfig := make([]DailyNotification, 0, len(config.DailyNotifications))
	for _, notification := range config.DailyNotifications {
		if notification.ID == notificationID {
			continue
		}

		filteredDailyConfig = append(filteredDailyConfig, notification)
	}

	config.DailyNotifications = filteredDailyConfig
	configState.Guilds[guildID] = config

	if err := store.saveNotificationScheduleState(notificationState); err != nil {
		return err
	}

	return store.saveConfigState(configState)
}

func (store *jsonNotificationConfigStore) MarkNotificationSent(_ context.Context, notificationID string, sentAt time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadNotificationScheduleState()
	if err != nil {
		return err
	}

	normalizedSentAt := sentAt.UTC()
	for index := range state.Notifications {
		notification := &state.Notifications[index]
		if notification.ID != notificationID {
			continue
		}

		nextNotificationAt, err := calculateNextNotificationAt(*notification, normalizedSentAt)
		if err != nil {
			return err
		}

		notification.NextNotificationAt = nextNotificationAt
		return store.saveNotificationScheduleState(state)
	}

	return errors.New("notificación no encontrada")
}

func (store *jsonNotificationConfigStore) RecalculateAllNextNotifications(_ context.Context, now time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	state, err := store.loadNotificationScheduleState()
	if err != nil {
		return err
	}

	normalizedNow := now.UTC()
	for index := range state.Notifications {
		notification := &state.Notifications[index]

		next, err := calculateNextFromBaseHour(*notification, normalizedNow)
		if err != nil {
			return err
		}

		notification.NextNotificationAt = next
	}

	return store.saveNotificationScheduleState(state)
}

func (store *jsonNotificationConfigStore) loadConfigState() (notificationConfigState, error) {
	content, err := os.ReadFile(store.configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return notificationConfigState{Guilds: make(map[string]NotificationConfig)}, nil
		}

		return notificationConfigState{}, err
	}

	var state notificationConfigState
	if err := json.Unmarshal(content, &state); err != nil {
		return notificationConfigState{}, err
	}

	if state.Guilds == nil {
		state.Guilds = make(map[string]NotificationConfig)
	}

	return state, nil
}

func (store *jsonNotificationConfigStore) loadNotificationScheduleState() (notificationScheduleState, error) {
	content, err := os.ReadFile(store.notificationsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return notificationScheduleState{Notifications: make([]ScheduledNotification, 0)}, nil
		}

		return notificationScheduleState{}, err
	}

	var state notificationScheduleState
	if err := json.Unmarshal(content, &state); err != nil {
		return notificationScheduleState{}, err
	}

	if state.Notifications == nil {
		state.Notifications = make([]ScheduledNotification, 0)
	}

	return state, nil
}

func (store *jsonNotificationConfigStore) saveConfigState(state notificationConfigState) error {
	if state.Guilds == nil {
		state.Guilds = make(map[string]NotificationConfig)
	}

	if err := os.MkdirAll(filepath.Dir(store.configFilePath), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(store.configFilePath, content, 0o644)
}

func (store *jsonNotificationConfigStore) saveNotificationScheduleState(state notificationScheduleState) error {
	if state.Notifications == nil {
		state.Notifications = make([]ScheduledNotification, 0)
	}

	if err := os.MkdirAll(filepath.Dir(store.notificationsFilePath), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(store.notificationsFilePath, content, 0o644)
}

func generateShortID() (string, error) {
	buffer := make([]byte, 3)
	if _, err := rand.Read(buffer); err != nil {
		return "", errors.New("no se pudo generar un id para la notificación")
	}

	return hex.EncodeToString(buffer), nil
}

func calculateInitialNextNotificationAt(baseHour string, everyMinutes int, now time.Time) (time.Time, error) {
	if everyMinutes <= 0 {
		return time.Time{}, errors.New("el valor every_minutes debe ser mayor a 0")
	}

	baseTime, err := time.Parse("15:04", baseHour)
	if err != nil {
		return time.Time{}, errors.New("el valor base_hour debe tener formato HH:MM (24h) en UTC")
	}

	normalizedNow := now.UTC()
	next := time.Date(
		normalizedNow.Year(),
		normalizedNow.Month(),
		normalizedNow.Day(),
		baseTime.Hour(),
		baseTime.Minute(),
		0,
		0,
		time.UTC,
	)

	interval := time.Duration(everyMinutes) * time.Minute
	for !next.After(normalizedNow) {
		next = next.Add(interval)
	}

	return next, nil
}

func calculateInitialDailyNextNotificationAt(baseHour string, now time.Time) (time.Time, error) {
	baseTime, err := time.Parse("15:04", baseHour)
	if err != nil {
		return time.Time{}, errors.New("el valor base_hour debe tener formato HH:MM (24h) en UTC")
	}

	normalizedNow := now.UTC()
	next := time.Date(
		normalizedNow.Year(),
		normalizedNow.Month(),
		normalizedNow.Day(),
		baseTime.Hour(),
		baseTime.Minute(),
		0,
		0,
		time.UTC,
	)

	for !next.After(normalizedNow) {
		next = next.Add(24 * time.Hour)
	}

	return next, nil
}

func calculateNextNotificationAt(notification ScheduledNotification, sentAt time.Time) (time.Time, error) {
	nextNotificationAt := notification.NextNotificationAt

	switch notification.Type {
	case "daily":
		for !nextNotificationAt.After(sentAt) {
			nextNotificationAt = nextNotificationAt.Add(24 * time.Hour)
		}
		return nextNotificationAt, nil
	case "byminutes":
		if notification.EveryMinutes <= 0 {
			return time.Time{}, errors.New("la notificación tiene un intervalo inválido")
		}

		interval := time.Duration(notification.EveryMinutes) * time.Minute
		for !nextNotificationAt.After(sentAt) {
			nextNotificationAt = nextNotificationAt.Add(interval)
		}
		return nextNotificationAt, nil
	default:
		return time.Time{}, errors.New("tipo de notificación no soportado")
	}
}

func calculateNextFromBaseHour(notification ScheduledNotification, now time.Time) (time.Time, error) {
	switch notification.Type {
	case "daily":
		return calculateInitialDailyNextNotificationAt(notification.BaseHour, now)
	case "byminutes":
		return calculateInitialNextNotificationAt(notification.BaseHour, notification.EveryMinutes, now)
	default:
		return time.Time{}, errors.New("tipo de notificación no soportado")
	}
}
