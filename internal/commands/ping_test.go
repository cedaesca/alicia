package commands

import (
	"context"
	"testing"

	"github.com/cedaesca/alicia/internal/discord"
)

func TestPingCommandExecute(t *testing.T) {
	command := NewPingCommand()

	t.Run("quotes user", func(t *testing.T) {
		response, err := command.Execute(context.Background(), discord.Interaction{UserID: "123456"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if response != "<@123456> Pong!" {
			t.Fatalf("unexpected response: %q", response)
		}
	})

	t.Run("fallback without user", func(t *testing.T) {
		response, err := command.Execute(context.Background(), discord.Interaction{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if response != "Pong!" {
			t.Fatalf("unexpected response: %q", response)
		}
	})
}

func TestPingCommandDefinition(t *testing.T) {
	definition := NewPingCommand().Definition()

	if definition.Name != "ping" {
		t.Fatalf("expected name ping, got %q", definition.Name)
	}

	if definition.Description == "" {
		t.Fatal("expected non-empty description")
	}
}
