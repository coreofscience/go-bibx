package collection

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/urfave/cli/v3"
)

func newDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm", "del"},
		Usage:     "Delete a collection by ID or name",
		ArgsUsage: "<id|name>",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "target",
				UsageText: "collection ID or name alias to delete",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			target := c.StringArg("target")
			if target == "" {
				return errors.New("collection ID or name argument is required")
			}

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
			col, err := colSvc.Resolve(ctx, target)
			if err != nil {
				if errors.Is(err, services.ErrCollectionNotFound) {
					return fmt.Errorf("collection not found: %s", target)
				}
				return fmt.Errorf("failed to resolve collection: %w", err)
			}

			if err := colSvc.Delete(ctx, col.ID); err != nil {
				return fmt.Errorf("failed to delete collection: %w", err)
			}

			if col.Name != nil && *col.Name != "" {
				fmt.Printf("Deleted collection %s (%s)\n", col.ID.String(), *col.Name)
			} else {
				fmt.Printf("Deleted collection %s\n", col.ID.String())
			}
			return nil
		},
	}
}
