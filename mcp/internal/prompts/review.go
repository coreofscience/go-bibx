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

//go:embed review.md
var reviewMarkdown string

var reviewTmpl = template.Must(template.New("review").Parse(reviewMarkdown))

// Review represents the literature review MCP prompt.
type Review struct{}

// NewReview creates a new [Review] prompt instance.
func NewReview() *Review {
	return &Review{}
}

// Prompt returns the [mcp.Prompt] definition for the review prompt.
func (p *Review) Prompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "review",
		Title:       "Conduct a Literature Review with Tree of Science",
		Description: "Structured workflow to perform a systematic literature review on a topic using the Tree of Science (roots, trunks, leaves) methodology and bibx MCP tools.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "topic",
				Title:       "Research Topic",
				Description: "The research topic, question, or keywords to review",
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

// Handle handles the prompts/get request for review.
func (p *Review) Handle(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var topic, target string
	if req.Params != nil && req.Params.Arguments != nil {
		topic = strings.TrimSpace(req.Params.Arguments["topic"])
		target = strings.TrimSpace(req.Params.Arguments["target"])
	}

	if topic == "" {
		return nil, errors.New("topic argument is required")
	}

	var buf bytes.Buffer
	err := reviewTmpl.Execute(&buf, struct {
		Topic  string
		Target string
	}{
		Topic:  topic,
		Target: target,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render review prompt: %w", err)
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Systematic literature review workflow for topic: %s", topic),
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

// Register registers the review prompt on the provided MCP server.
func (p *Review) Register(server *mcp.Server) {
	server.AddPrompt(p.Prompt(), p.Handle)
}
