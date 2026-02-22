package discord

import (
	"errors"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type fakeSession struct {
	openErr     error
	closeErr    error
	sendErr     error
	registerErr error
	respondErr  error

	sentChannelID         string
	sentContent           string
	registeredName        string
	registeredDescription string

	handler            func(message *discordgo.MessageCreate)
	interactionHandler func(interaction *discordgo.InteractionCreate)
}

func (session *fakeSession) Open() error {
	return session.openErr
}

func (session *fakeSession) Close() error {
	return session.closeErr
}

func (session *fakeSession) AddMessageCreateHandler(handler func(message *discordgo.MessageCreate)) {
	session.handler = handler
}

func (session *fakeSession) AddInteractionCreateHandler(handler func(interaction *discordgo.InteractionCreate)) {
	session.interactionHandler = handler
}

func (session *fakeSession) ApplicationCommandCreate(name, description string) (string, error) {
	session.registeredName = name
	session.registeredDescription = description

	if session.registerErr != nil {
		return "", session.registerErr
	}

	return "command-id-1", nil
}

func (session *fakeSession) InteractionRespond(interaction *discordgo.Interaction, content string) error {
	session.sentContent = content
	return session.respondErr
}

func (session *fakeSession) ChannelMessageSend(channelID, content string) error {
	session.sentChannelID = channelID
	session.sentContent = content
	return session.sendErr
}

func TestNewDiscordGoClient(t *testing.T) {
	client, err := NewDiscordGoClient("test-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestNewClientAlias(t *testing.T) {
	client, err := NewClient("test-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}
}

func TestDiscordGoClientOpen(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &discordGoClient{session: &fakeSession{}}

		if err := client.Open(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("open failed")
		client := &discordGoClient{session: &fakeSession{openErr: expectedErr}}

		err := client.Open()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestDiscordGoClientClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &discordGoClient{session: &fakeSession{}}

		if err := client.Close(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("close failed")
		client := &discordGoClient{session: &fakeSession{closeErr: expectedErr}}

		err := client.Close()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestDiscordGoClientSendMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		session := &fakeSession{}
		client := &discordGoClient{session: session}

		err := client.SendMessage("channel-1", "hello")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if session.sentChannelID != "channel-1" || session.sentContent != "hello" {
			t.Fatalf("unexpected send args: channel=%q content=%q", session.sentChannelID, session.sentContent)
		}
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("send failed")
		session := &fakeSession{sendErr: expectedErr}
		client := &discordGoClient{session: session}

		err := client.SendMessage("channel-1", "hello")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestDiscordGoClientAddMessageCreateHandler(t *testing.T) {
	t.Run("maps message", func(t *testing.T) {
		session := &fakeSession{}
		client := &discordGoClient{session: session}

		var received Message
		client.AddMessageCreateHandler(func(message Message) {
			received = message
		})

		if session.handler == nil {
			t.Fatal("expected handler to be registered")
		}

		session.handler(&discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg-1",
				ChannelID: "chan-1",
				GuildID:   "guild-1",
				Author:    &discordgo.User{ID: "user-1"},
				Content:   "hello",
			},
		})

		if received.ID != "msg-1" || received.ChannelID != "chan-1" || received.GuildID != "guild-1" || received.AuthorID != "user-1" || received.Content != "hello" {
			t.Fatalf("unexpected mapped message: %+v", received)
		}
	})

	t.Run("nil author handled", func(t *testing.T) {
		session := &fakeSession{}
		client := &discordGoClient{session: session}

		var received Message
		client.AddMessageCreateHandler(func(message Message) {
			received = message
		})

		session.handler(&discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg-2",
				ChannelID: "chan-2",
				GuildID:   "guild-2",
				Author:    nil,
				Content:   "no author",
			},
		})

		if received.AuthorID != "" {
			t.Fatalf("expected empty author id, got %q", received.AuthorID)
		}
	})
}

func TestDiscordGoClientRegisterGlobalCommand(t *testing.T) {
	session := &fakeSession{}
	client := &discordGoClient{session: session}

	commandID, err := client.RegisterGlobalCommand(SlashCommand{Name: "ping", Description: "Replies with Pong!"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if commandID != "command-id-1" {
		t.Fatalf("expected command id command-id-1, got %q", commandID)
	}

	if session.registeredName != "ping" || session.registeredDescription != "Replies with Pong!" {
		t.Fatalf("unexpected command registration payload: %q / %q", session.registeredName, session.registeredDescription)
	}
}

func TestDiscordGoClientRespondToInteraction(t *testing.T) {
	t.Run("missing raw interaction", func(t *testing.T) {
		client := &discordGoClient{session: &fakeSession{}}

		err := client.RespondToInteraction(Interaction{}, "hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		session := &fakeSession{}
		client := &discordGoClient{session: session}

		err := client.RespondToInteraction(Interaction{raw: &discordgo.Interaction{}}, "hello")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if session.sentContent != "hello" {
			t.Fatalf("expected response content hello, got %q", session.sentContent)
		}
	})
}
