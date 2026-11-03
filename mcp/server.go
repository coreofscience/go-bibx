// Package mcp builds the bibx Model Context Protocol server.
//
// The server exposes the bibx tools, resources and prompts on top of the
// given services. It is transport agnostic: run it over stdio, or serve it
// over HTTP with the SDK's streamable HTTP handler.
package mcp

import (
	_ "embed"
	"log/slog"
	"uuid"

	"github.com/coreofscience/go-bibx/internal/utils"
	"github.com/coreofscience/go-bibx/mcp/internal/prompts"
	"github.com/coreofscience/go-bibx/mcp/internal/resources"
	"github.com/coreofscience/go-bibx/mcp/internal/tools"
	"github.com/coreofscience/go-bibx/services"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed instructions.txt
var instructions string

type options struct {
	logger  *slog.Logger
	version string
}

// Option configures the server built by [NewServer].
type Option func(*options)

// WithLogger sets the logger used by the server. It defaults to [slog.Default].
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithVersion sets the version the server reports to clients. It defaults to
// the version of the go-bibx module in the build.
func WithVersion(version string) Option {
	return func(o *options) {
		o.version = version
	}
}

// NewServer creates an MCP server with every bibx tool, resource and prompt
// registered on top of the given services.
func NewServer(
	collectionService services.CollectionService,
	analysisService services.AnalysisService,
	opts ...Option,
) *sdk.Server {
	o := &options{
		logger:  slog.Default(),
		version: utils.ModuleVersion(),
	}
	for _, opt := range opts {
		opt(o)
	}

	server := sdk.NewServer(
		&sdk.Implementation{
			Name:        "bibx",
			Title:       "bibx",
			Description: "Maps the literature on any research topic from OpenAlex with the Tree of Science, to find its foundational, structural and current works, search them and export citations",
			Version:     o.version,
			WebsiteURL:  "https://bibx.coreofscience.org",
		},
		&sdk.ServerOptions{
			Instructions: instructions,
			Logger:       o.logger,
			Capabilities: &sdk.ServerCapabilities{
				Prompts: &sdk.PromptCapabilities{
					ListChanged: false,
				},
				Resources: &sdk.ResourceCapabilities{
					ListChanged: false,
					Subscribe:   false,
				},
				Tools: &sdk.ToolCapabilities{
					ListChanged: false,
				},
			},
			GetSessionID: func() string {
				return uuid.NewV7().String()
			},
		},
	)

	tools.Register(server, collectionService, analysisService)
	resources.Register(server, collectionService, analysisService)
	prompts.Register(server)

	return server
}
