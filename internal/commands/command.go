package commands

import (
	"context"

	"github.com/cedaesca/alicia/internal/discord"
)

type Command interface {
	Definition() discord.SlashCommand
	Execute(ctx context.Context, interaction discord.Interaction) (string, error)
}

func All() []Command {
	return []Command{
		NewPingCommand(),
	}
}
