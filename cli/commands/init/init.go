package init

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/coreofscience/go-bibx/cli/services"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/urfave/cli/v3"
)

func New() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new project with the default template",
		Action: func(ctx context.Context, c *cli.Command) error {
			rootDir, ok := utils.GetRootDir(ctx)
			if !ok {
				return errors.New("root directory not set in context")
			}
			scaffoldService := services.NewDefaultScaffoldServiceWithConfig(
				&services.DefaultScaffoldServiceConfig{
					ScaffoldPath: rootDir,
				},
			)
			if err := scaffoldService.Scaffold(ctx); err != nil {
				return fmt.Errorf("failed to scaffold project: %w", err)
			}
			slog.Info("project scaffolded successfully")
			return nil
		},
	}
}
