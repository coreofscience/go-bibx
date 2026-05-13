package commands

import (
	"github.com/coreofscience/go-bibx/cli/commands/inquire"
	"github.com/coreofscience/go-bibx/cli/commands/query"
	"github.com/coreofscience/go-bibx/cli/commands/setup"
	"github.com/coreofscience/go-bibx/cli/commands/view"
	"github.com/urfave/cli/v3"
)

func New() []*cli.Command {
	return []*cli.Command{
		inquire.New(),
		view.New(),
		query.New(),
		setup.New(),
	}
}
