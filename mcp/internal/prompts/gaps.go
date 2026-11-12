package prompts

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed gaps.md
var gapsMarkdown string

var gapsTmpl = template.Must(template.New("gaps").Parse(gapsMarkdown))

// Gaps represents the research gap analysis MCP prompt.
type Gaps struct{}

// NewGaps creates a new [Gaps] prompt instance.
func NewGaps() *Gaps {
	return &Gaps{}
}

// Prompt returns the [mcp.Prompt] definition for the gaps prompt.
func (p *Gaps) Prompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "gaps",
		Title:       "Find Research Gaps with Tree of Science",
		Description: "Systematic methodology to identify unexplored gaps, disconnected lineages, and frontier opportunities in a field using Tree of Science network analysis.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "topic",
				Title:       "Research Topic",
				Description: "The research field or problem domain to investigate for gaps",
				Required:    true,
			},
			{
				Name:        "target",
				Title:       "Collection Target",
				Description: "Existing collection UUID or name alias (optional)",
				Required:    false,
			},
		},
	}
}

// Handle handles the prompts/get request for gaps.
func (p *Gaps) Handle(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var topic, target string
	if req.Params != nil && req.Params.Arguments != nil {
		topic = strings.TrimSpace(req.Params.Arguments["topic"])
		target = strings.TrimSpace(req.Params.Arguments["target"])
	}

	if topic == "" {
		return nil, errors.New("topic argument is required")
	}

	var buf bytes.Buffer
	err := gapsTmpl.Execute(&buf, struct {
		Topic  string
		Target string
	}{
		Topic:  topic,
		Target: target,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render gaps prompt: %w", err)
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Research gap analysis workflow for field: %s", topic),
		Messages: []*mcp.PromptMessage{
			{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: strings.TrimSpace(buf.String()),
				},
			},
		},
	}, nil
}

// Register registers the gaps prompt on the provided MCP server.
func (p *Gaps) Register(server *mcp.Server) {
	server.AddPrompt(p.Prompt(), p.Handle)
}
