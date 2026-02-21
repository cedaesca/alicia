package app

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/cedaesca/alicia/internal/discord"
)

type fakeDiscordClient struct {
	openErr  error
	closeErr error
	closeCh  chan struct{}
}

func (client *fakeDiscordClient) Open() error {
	return client.openErr
}

func (client *fakeDiscordClient) Close() error {
	if client.closeCh != nil {
		<-client.closeCh
	}

	return client.closeErr
}

func (client *fakeDiscordClient) AddMessageCreateHandler(handler discord.MessageCreateHandler) {
}

func (client *fakeDiscordClient) SendMessage(channelID, content string) error {
	return nil
}

func TestNewApplication(t *testing.T) {
	t.Run("uses background when context is nil", func(t *testing.T) {
		application, err := NewApplication(nil, "test-token")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if application.Context() == nil {
			t.Fatal("expected context to be initialized")
		}

		if application.Logger() == nil {
			t.Fatal("expected logger to be initialized")
		}

		if application.DiscordClient() == nil {
			t.Fatal("expected discord client to be initialized")
		}
	})

	t.Run("fails when token is missing", func(t *testing.T) {
		application, err := NewApplication(context.Background(), "   ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if application != nil {
			t.Fatal("expected nil application")
		}
	})
}

func TestApplicationRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{},
		}

		if err := application.Run(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error from discord client", func(t *testing.T) {
		expectedErr := errors.New("open failed")
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{openErr: expectedErr},
		}

		err := application.Run()
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestApplicationShutdown(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{},
		}

		if err := application.Shutdown(context.Background()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("error from discord client", func(t *testing.T) {
		expectedErr := errors.New("close failed")
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{closeErr: expectedErr},
		}

		err := application.Shutdown(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("context cancelled before close completes", func(t *testing.T) {
		closeCh := make(chan struct{})
		application := &Application{
			ctx:           context.Background(),
			logger:        log.New(io.Discard, "", 0),
			discordClient: &fakeDiscordClient{closeCh: closeCh},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := application.Shutdown(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}

		close(closeCh)
	})
}
