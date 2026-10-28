package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"uuid"

	"github.com/coreofscience/go-bibx/clients/openalex"
	"github.com/coreofscience/go-bibx/internal/mcp/resources"
	"github.com/coreofscience/go-bibx/internal/mcp/tools"
	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/repository"
	"github.com/coreofscience/go-bibx/services"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"
)

// New creates the mcp server cli command.
func New() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "start mcp server",
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
			anaSvc := services.NewAnalysisService(db, openalex.NewRestyClient())

			server := mcp.NewServer(
				&mcp.Implementation{
					Name:        "bibx",
					Title:       "bibx",
					Description: "bibx provides capabilities for managing bibliographic collections",
					Version:     "0.2.0",
					WebsiteURL:  "https://bibx.coreofscience.org",
				},
				&mcp.ServerOptions{
					Instructions: "use the provided tools and resources to manage the users bibliography, use other methods to gather more information about articles",
					Logger:       slog.Default(),
					Capabilities: &mcp.ServerCapabilities{
						Prompts: &mcp.PromptCapabilities{
							ListChanged: false,
						},
						Resources: &mcp.ResourceCapabilities{
							ListChanged: false,
							Subscribe:   false,
						},
						Tools: &mcp.ToolCapabilities{
							ListChanged: false,
						},
					},
					GetSessionID: func() string {
						return uuid.NewV7().String()
					},
				},
			)

			tools.Register(server, colSvc, anaSvc)
			resources.Register(server, colSvc, anaSvc)

			if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
				return fmt.Errorf("failed to run mcp server: %w", err)
			}

			return nil
		},
	}
}
