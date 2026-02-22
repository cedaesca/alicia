package discord

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

type discordSession interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler func(message *discordgo.MessageCreate))
	AddInteractionCreateHandler(handler func(interaction *discordgo.InteractionCreate))
	ApplicationCommandCreate(name, description string) (string, error)
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

func (discordSession *discordGoSession) ApplicationCommandCreate(name, description string) (string, error) {
	if discordSession.session.State == nil || discordSession.session.State.User == nil {
		return "", errors.New("discord session user state is not initialized")
	}

	command, err := discordSession.session.ApplicationCommandCreate(
		discordSession.session.State.User.ID,
		"",
		&discordgo.ApplicationCommand{
			Name:        name,
			Description: description,
		},
	)
	if err != nil {
		return "", err
	}

	return command.ID, nil
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
}

type Interaction struct {
	ID          string
	CommandName string
	ChannelID   string
	GuildID     string
	UserID      string
	raw         *discordgo.Interaction
}

type InteractionCreateHandler func(interaction Interaction)

type Client interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler MessageCreateHandler)
	AddInteractionCreateHandler(handler InteractionCreateHandler)
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
			raw:         interactionCreate.Interaction,
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
	return client.session.ApplicationCommandCreate(command.Name, command.Description)
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
