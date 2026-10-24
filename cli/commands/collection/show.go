package collection

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/storage"
	"github.com/urfave/cli/v3"
)

func newShowCommand() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Aliases:   []string{"info", "get"},
		Usage:     "Display details of a collection by ID or name",
		ArgsUsage: "[<id|name>]",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "target",
				UsageText: "collection ID or name alias (defaults to latest)",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			dbPath, ok := utils.GetDBPath(ctx)
			if !ok {
				return errors.New("database path not set")
			}

			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = store.Close()
			}()

			target := c.StringArg("target")
			col, err := storage.ResolveCollection(ctx, store, target)
			if err != nil {
				if errors.Is(err, storage.ErrCollectionNotFound) {
					if target == "" {
						return errors.New("no collections found")
					}
					return fmt.Errorf("collection not found: %s", target)
				}
				return fmt.Errorf("failed to resolve collection: %w", err)
			}

			name := "-"
			if col.Name != nil && *col.Name != "" {
				name = *col.Name
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "ID:\t%s\n", col.ID.String())
			_, _ = fmt.Fprintf(w, "Name:\t%s\n", name)
			_, _ = fmt.Fprintf(w, "Query:\t%s\n", col.Query)
			_, _ = fmt.Fprintf(w, "Limit:\t%d\n", col.ResultLimit)
			_, _ = fmt.Fprintf(w, "Articles:\t%d\n", col.ArticleCount)
			_, _ = fmt.Fprintf(w, "Created:\t%s\n", col.CreatedAt.Format("2006-01-02 15:04:05"))
			_, _ = fmt.Fprintf(w, "Updated:\t%s\n", col.UpdatedAt.Format("2006-01-02 15:04:05"))
			return w.Flush()
		},
	}
}
