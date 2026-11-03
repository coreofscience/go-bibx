package collection

import (
	"github.com/urfave/cli/v3"
)

// New returns the collection command with its subcommands (list, get, delete, export).
func New() *cli.Command {
	return &cli.Command{
		Name:  "collection",
		Usage: "Manage bibliographic inquiry collections",
		Commands: []*cli.Command{
			newListCommand(),
			newGetCommand(),
			newDeleteCommand(),
			newExportCommand(),
		},
	}
}
