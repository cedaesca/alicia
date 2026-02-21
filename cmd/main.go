package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cedaesca/alicia/internal/app"
)

var (
	Token string
)

func parseAndValidateConfig() error {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()

	if strings.TrimSpace(Token) == "" {
		return errors.New("missing bot token: pass it with -t")
	}

	return nil
}

func gracefulShutdown(application *app.Application, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	application.Logger().Println("Shutdown signal received")
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := application.Shutdown(ctx); err != nil {
		log.Printf("Alicia forced to shutdown with error: %v", err)
	}

	done <- true
}

func main() {
	log.Println("Starting up Alicia...")

	if err := parseAndValidateConfig(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx := context.Background()
	application, err := app.NewApplication(ctx, Token)
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("failed to start application: %v", err)
	}

	done := make(chan bool, 1)

	go gracefulShutdown(application, done)

	<-done
}
