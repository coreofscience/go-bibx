// Package mcp builds the bibx Model Context Protocol server.
//
// The server exposes the bibx tools, resources and prompts on top of the
// given services. It is transport agnostic: run it over stdio, or serve it
// over HTTP with the SDK's streamable HTTP handler.
package mcp

import (
	_ "embed"
	"log/slog"
	"runtime/debug"
	"uuid"

	"github.com/coreofscience/go-bibx/mcp/internal/prompts"
	"github.com/coreofscience/go-bibx/mcp/internal/resources"
	"github.com/coreofscience/go-bibx/mcp/internal/tools"
	"github.com/coreofscience/go-bibx/services"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const modulePath = "github.com/coreofscience/go-bibx"

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
		version: moduleVersion(),
	}
	for _, opt := range opts {
		opt(o)
	}

	server := sdk.NewServer(
		&sdk.Implementation{
			Name:        "bibx",
			Title:       "bibx",
			Description: "bibx provides capabilities for managing bibliographic collections",
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

// moduleVersion reports the version of the go-bibx module in the running
// binary, whether it is the main module or a dependency.
func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	if info.Main.Path == modulePath {
		return info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return dep.Version
		}
	}
	return "(devel)"
}
