package collection

import (
	"github.com/urfave/cli/v3"
)

// New returns the collection command with its subcommands (list, show, delete).
func New() *cli.Command {
	return &cli.Command{
		Name:    "collection",
		Aliases: []string{"collections", "col"},
		Usage:   "Manage bibliographic inquiry collections",
		Commands: []*cli.Command{
			newListCommand(),
			newShowCommand(),
			newDeleteCommand(),
			newExportCommand(),
		},
	}
}
