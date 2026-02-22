package commands

import (
	"context"
	"fmt"

	"github.com/cedaesca/alicia/internal/discord"
)

type pingCommand struct{}

func NewPingCommand() Command {
	return &pingCommand{}
}

func (command *pingCommand) Definition() discord.SlashCommand {
	return discord.SlashCommand{
		Name:        "ping",
		Description: "Responde con Pong!",
	}
}

func (command *pingCommand) Execute(_ context.Context, interaction discord.Interaction) (string, error) {
	if interaction.UserID == "" {
		return "Pong!", nil
	}

	return fmt.Sprintf("<@%s> Pong!", interaction.UserID), nil
}
