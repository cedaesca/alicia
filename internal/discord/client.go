package discord

import (
	"github.com/bwmarrin/discordgo"
)

type discordSession interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler func(message *discordgo.MessageCreate))
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

type Client interface {
	Open() error
	Close() error
	AddMessageCreateHandler(handler MessageCreateHandler)
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

func (client *discordGoClient) SendMessage(channelID, content string) error {
	return client.session.ChannelMessageSend(channelID, content)
}
