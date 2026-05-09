package commands

import (
	"github.com/coreofscience/go-bibx/internal/commands/inquire"
	"github.com/coreofscience/go-bibx/internal/commands/view"
	"github.com/urfave/cli/v3"
)

func New() []*cli.Command {
	return []*cli.Command{
		inquire.New(),
		view.New(),
	}
}
