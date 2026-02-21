package app

import (
	"context"
	"log"
	"os"

	"github.com/cedaesca/alicia/internal/discord"
)

type Application struct {
	ctx           context.Context
	logger        *log.Logger
	discordClient discord.Client
}

func NewApplication(ctx context.Context, token string) (*Application, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := log.New(os.Stdout, "[alicia] ", log.LstdFlags)

	discordClient, err := discord.NewDiscordGoClient(token)
	if err != nil {
		return nil, err
	}

	return &Application{
		ctx:           ctx,
		logger:        logger,
		discordClient: discordClient,
	}, nil
}

func (application *Application) Context() context.Context {
	return application.ctx
}

func (application *Application) Logger() *log.Logger {
	return application.logger
}

func (application *Application) DiscordClient() discord.Client {
	return application.discordClient
}

func (application *Application) Run() error {
	application.logger.Println("Starting Discord client")

	if err := application.discordClient.Open(); err != nil {
		return err
	}

	application.logger.Println("Discord client is running")
	return nil
}

func (application *Application) Shutdown(ctx context.Context) error {
	application.logger.Println("Shutting down Discord client")

	errChan := make(chan error, 1)
	go func() {
		errChan <- application.discordClient.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err == nil {
			application.logger.Println("Discord client shutdown complete")
		}

		return err
	}
}
