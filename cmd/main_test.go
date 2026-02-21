package main

import (
	"flag"
	"testing"
)

func TestParseAndValidateConfigFrom(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

		token, err := parseAndValidateConfigFrom(flagSet, []string{"-t", "abc123"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if token != "abc123" {
			t.Fatalf("expected token abc123, got %q", token)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

		token, err := parseAndValidateConfigFrom(flagSet, []string{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "missing bot token: pass it with -t" {
			t.Fatalf("unexpected error: %v", err)
		}

		if token != "" {
			t.Fatalf("expected empty token, got %q", token)
		}
	})

	t.Run("invalid flag", func(t *testing.T) {
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

		_, err := parseAndValidateConfigFrom(flagSet, []string{"-unknown", "value"})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})
}
