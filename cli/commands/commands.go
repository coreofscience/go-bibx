package commands

import (
	initcmd "github.com/coreofscience/go-bibx/cli/commands/init"
	"github.com/coreofscience/go-bibx/cli/commands/inquire"
	"github.com/coreofscience/go-bibx/cli/commands/query"
	"github.com/coreofscience/go-bibx/cli/commands/search"
	"github.com/coreofscience/go-bibx/cli/commands/setup"
	"github.com/coreofscience/go-bibx/cli/commands/view"
	"github.com/urfave/cli/v3"
)

func New() []*cli.Command {
	return []*cli.Command{
		initcmd.New(),
		inquire.New(),
		query.New(),
		search.New(),
		setup.New(),
		view.New(),
	}
}
