package discord

import (
	"github.com/bwmarrin/discordgo"
)

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
	session *discordgo.Session
}

func NewDiscordGoClient(token string) (Client, error) {
	session, err := discordgo.New("Bot " + token)

	if err != nil {
		return nil, err
	}

	return &discordGoClient{session: session}, nil
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
	client.session.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageCreate) {
		handler(Message{
			ID:        message.ID,
			ChannelID: message.ChannelID,
			GuildID:   message.GuildID,
			AuthorID:  message.Author.ID,
			Content:   message.Content,
		})
	})
}

func (client *discordGoClient) SendMessage(channelID, content string) error {
	_, err := client.session.ChannelMessageSend(channelID, content)
	return err
}
