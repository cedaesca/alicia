package discord

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type discordSession interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler func(message *discordgo.MessageCreate))
	AddInteractionCreateHandler(handler func(interaction *discordgo.InteractionCreate))
	ApplicationCommandCreate(command SlashCommand) (string, error)
	ApplicationCommands() ([]RegisteredSlashCommand, error)
	InteractionRespond(interaction *discordgo.Interaction, content string) error
	ChannelMessageSend(channelID, content string) error
}

type discordGoSession struct {
	session *discordgo.Session
}

func (discordSession *discordGoSession) Open() error {
	return discordSession.session.Open()
}

func (discordSession *discordGoSession) Close() error {
	return discordSession.session.Close()
}

func (discordSession *discordGoSession) AddMessageCreateHandler(handler func(message *discordgo.MessageCreate)) {
	discordSession.session.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageCreate) {
		handler(message)
	})
}

func (discordSession *discordGoSession) AddInteractionCreateHandler(handler func(interaction *discordgo.InteractionCreate)) {
	discordSession.session.AddHandler(func(_ *discordgo.Session, interaction *discordgo.InteractionCreate) {
		handler(interaction)
	})
}

func (discordSession *discordGoSession) ApplicationCommandCreate(command SlashCommand) (string, error) {
	if discordSession.session.State == nil || discordSession.session.State.User == nil {
		return "", errors.New("discord session user state is not initialized")
	}

	options := make([]*discordgo.ApplicationCommandOption, 0, len(command.Options))
	for _, option := range command.Options {
		options = append(options, &discordgo.ApplicationCommandOption{
			Type:        toDiscordOptionType(option.Type),
			Name:        option.Name,
			Description: option.Description,
			Required:    option.Required,
		})
	}

	createdCommand, err := discordSession.session.ApplicationCommandCreate(
		discordSession.session.State.User.ID,
		"",
		&discordgo.ApplicationCommand{
			Name:        command.Name,
			Description: command.Description,
			Options:     options,
		},
	)
	if err != nil {
		return "", err
	}

	return createdCommand.ID, nil
}

func (discordSession *discordGoSession) ApplicationCommands() ([]RegisteredSlashCommand, error) {
	if discordSession.session.State == nil || discordSession.session.State.User == nil {
		return nil, errors.New("discord session user state is not initialized")
	}

	commands, err := discordSession.session.ApplicationCommands(discordSession.session.State.User.ID, "")
	if err != nil {
		return nil, err
	}

	registered := make([]RegisteredSlashCommand, 0, len(commands))
	for _, command := range commands {
		registered = append(registered, RegisteredSlashCommand{ID: command.ID, Name: command.Name})
	}

	return registered, nil
}

func (discordSession *discordGoSession) InteractionRespond(interaction *discordgo.Interaction, content string) error {
	return discordSession.session.InteractionRespond(
		interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content},
		},
	)
}

func (discordSession *discordGoSession) ChannelMessageSend(channelID, content string) error {
	_, err := discordSession.session.ChannelMessageSend(channelID, content)
	return err
}

type Message struct {
	ID        string
	ChannelID string
	GuildID   string
	AuthorID  string
	Content   string
}

type MessageCreateHandler func(message Message)

type SlashCommand struct {
	Name        string
	Description string
	Options     []SlashCommandOption
}

type RegisteredSlashCommand struct {
	ID   string
	Name string
}

type SlashCommandOptionType string

const (
	SlashCommandOptionTypeString  SlashCommandOptionType = "string"
	SlashCommandOptionTypeInteger SlashCommandOptionType = "integer"
	SlashCommandOptionTypeChannel SlashCommandOptionType = "channel"
	SlashCommandOptionTypeRole    SlashCommandOptionType = "role"
)

type SlashCommandOption struct {
	Name        string
	Description string
	Type        SlashCommandOptionType
	Required    bool
}

type Interaction struct {
	ID          string
	CommandName string
	ChannelID   string
	GuildID     string
	UserID      string
	Options     map[string]string
	raw         *discordgo.Interaction
}

type InteractionCreateHandler func(interaction Interaction)

type Client interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler MessageCreateHandler)
	AddInteractionCreateHandler(handler InteractionCreateHandler)
	ListGlobalCommands() ([]RegisteredSlashCommand, error)
	RegisterGlobalCommand(command SlashCommand) (string, error)
	RespondToInteraction(interaction Interaction, content string) error
	SendMessage(channelID, content string) error
}

type discordGoClient struct {
	session discordSession
}

func NewDiscordGoClient(token string) (Client, error) {
	session, err := discordgo.New("Bot " + token)

	if err != nil {
		return nil, err
	}

	return &discordGoClient{session: &discordGoSession{session: session}}, nil
}

func NewClient(token string) (Client, error) {
	return NewDiscordGoClient(token)
}

func (client *discordGoClient) Open() error {
	return client.session.Open()
}

func (client *discordGoClient) Close() error {
	return client.session.Close()
}

func (client *discordGoClient) AddMessageCreateHandler(handler MessageCreateHandler) {
	client.session.AddMessageCreateHandler(func(message *discordgo.MessageCreate) {
		authorID := ""
		if message.Author != nil {
			authorID = message.Author.ID
		}

		handler(Message{
			ID:        message.ID,
			ChannelID: message.ChannelID,
			GuildID:   message.GuildID,
			AuthorID:  authorID,
			Content:   message.Content,
		})
	})
}

func (client *discordGoClient) AddInteractionCreateHandler(handler InteractionCreateHandler) {
	client.session.AddInteractionCreateHandler(func(interactionCreate *discordgo.InteractionCreate) {
		if interactionCreate.Type != discordgo.InteractionApplicationCommand {
			return
		}

		interaction := Interaction{
			ID:          interactionCreate.ID,
			CommandName: interactionCreate.ApplicationCommandData().Name,
			ChannelID:   interactionCreate.ChannelID,
			GuildID:     interactionCreate.GuildID,
			Options:     make(map[string]string),
			raw:         interactionCreate.Interaction,
		}

		for _, option := range interactionCreate.ApplicationCommandData().Options {
			interaction.Options[option.Name] = optionValueToString(option)
		}

		if interactionCreate.Member != nil && interactionCreate.Member.User != nil {
			interaction.UserID = interactionCreate.Member.User.ID
		} else if interactionCreate.User != nil {
			interaction.UserID = interactionCreate.User.ID
		}

		handler(interaction)
	})
}

func (client *discordGoClient) RegisterGlobalCommand(command SlashCommand) (string, error) {
	return client.session.ApplicationCommandCreate(command)
}

func (client *discordGoClient) ListGlobalCommands() ([]RegisteredSlashCommand, error) {
	return client.session.ApplicationCommands()
}

func (client *discordGoClient) RespondToInteraction(interaction Interaction, content string) error {
	if interaction.raw == nil {
		return errors.New("interaction payload is empty")
	}

	return client.session.InteractionRespond(interaction.raw, content)
}

func (client *discordGoClient) SendMessage(channelID, content string) error {
	return client.session.ChannelMessageSend(channelID, content)
}

func toDiscordOptionType(optionType SlashCommandOptionType) discordgo.ApplicationCommandOptionType {
	switch optionType {
	case SlashCommandOptionTypeInteger:
		return discordgo.ApplicationCommandOptionInteger
	case SlashCommandOptionTypeChannel:
		return discordgo.ApplicationCommandOptionChannel
	case SlashCommandOptionTypeRole:
		return discordgo.ApplicationCommandOptionRole
	default:
		return discordgo.ApplicationCommandOptionString
	}
}

func optionValueToString(option *discordgo.ApplicationCommandInteractionDataOption) string {
	if option == nil {
		return ""
	}

	switch value := option.Value.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case bool:
		if value {
			return "true"
		}

		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}
