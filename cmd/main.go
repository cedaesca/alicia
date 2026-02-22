package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
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
	token, err := parseAndValidateConfigFrom(flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}

	Token = token
	return nil
}

func parseAndValidateConfigFrom(flagSet *flag.FlagSet, args []string) (string, error) {
	if flagSet == nil {
		return "", errors.New("missing flag set")
	}

	var token string
	flagSet.StringVar(&token, "t", "", "Bot Token")

	if err := flagSet.Parse(args); err != nil {
		return "", err
	}

	if strings.TrimSpace(token) == "" {
		return "", errors.New("missing bot token: pass it with -t")
	}

	return token, nil
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
