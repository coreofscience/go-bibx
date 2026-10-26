package collection

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/urfave/cli/v3"
)

func newListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all collections in the database",
		Action: func(ctx context.Context, c *cli.Command) error {
			dbPath, ok := utils.GetDBPath(ctx)
			if !ok {
				return errors.New("database path not set")
			}

			db, err := repository.OpenAndMigrateSQLite(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			colSvc := services.NewCollectionService(db)
			collections, err := colSvc.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list collections: %w", err)
			}

			if len(collections) == 0 {
				fmt.Println("No collections found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tARTICLES\tQUERY\tCREATED")
			for _, col := range collections {
				name := "-"
				if col.Name != nil && *col.Name != "" {
					name = *col.Name
				}
				q := col.Query
				if len(q) > 40 {
					q = q[:37] + "..."
				}
				created := col.CreatedAt.Local().Format("2006-01-02 15:04")
				_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", col.ID.String(), name, col.ArticleCount, q, created)
			}
			return w.Flush()
		},
	}
}
