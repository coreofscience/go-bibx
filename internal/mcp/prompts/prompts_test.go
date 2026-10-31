package prompts_test

import (
	"strings"
	"testing"

	"github.com/coreofscience/go-bibx/internal/mcp/prompts"
	mcptest "github.com/coreofscience/go-bibx/internal/mcp/test"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "0.1.0"}, nil)
	prompts.Register(server)
	return mcptest.Connect(t, server)
}

func TestListPrompts(t *testing.T) {
	client := setupTestServer(t)

	res, err := client.ListPrompts(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Prompts, 2)

	promptsByName := make(map[string]*mcp.Prompt)
	for _, p := range res.Prompts {
		promptsByName[p.Name] = p
	}

	t.Run("review prompt metadata", func(t *testing.T) {
		p, exists := promptsByName["review"]
		require.True(t, exists)
		assert.Equal(t, "review", p.Name)
		assert.Equal(t, "Conduct a Literature Review with Tree of Science", p.Title)
		assert.Contains(t, p.Description, "systematic literature review")
		require.Len(t, p.Arguments, 2)

		assert.Equal(t, "topic", p.Arguments[0].Name)
		assert.True(t, p.Arguments[0].Required)
		assert.Equal(t, "target", p.Arguments[1].Name)
		assert.False(t, p.Arguments[1].Required)
	})

	t.Run("gaps prompt metadata", func(t *testing.T) {
		p, exists := promptsByName["gaps"]
		require.True(t, exists)
		assert.Equal(t, "gaps", p.Name)
		assert.Equal(t, "Find Research Gaps with Tree of Science", p.Title)
		assert.Contains(t, p.Description, "Tree of Science network analysis")
		require.Len(t, p.Arguments, 2)

		assert.Equal(t, "topic", p.Arguments[0].Name)
		assert.True(t, p.Arguments[0].Required)
		assert.Equal(t, "target", p.Arguments[1].Name)
		assert.False(t, p.Arguments[1].Required)
	})
}

func TestGetPromptReview(t *testing.T) {
	client := setupTestServer(t)

	t.Run("success with topic and target", func(t *testing.T) {
		res, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "review",
			Arguments: map[string]string{
				"topic":  "quantum computing error mitigation",
				"target": "qc-collection",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Description, "quantum computing error mitigation")
		require.Len(t, res.Messages, 1)

		msg := res.Messages[0]
		assert.Equal(t, mcp.Role("user"), msg.Role)

		textContent, ok := msg.Content.(*mcp.TextContent)
		require.True(t, ok)
		text := textContent.Text

		assert.Contains(t, text, "quantum computing error mitigation")
		assert.Contains(t, text, "**Target Collection**: qc-collection")
		assert.Contains(t, text, "bibx:/summary")
		assert.Contains(t, text, `tos(category="root"`)
		assert.Contains(t, text, "not by whether it mentions the query")
		assert.Contains(t, text, `tos(category="trunk"`)
		assert.Contains(t, text, `tos(category="leaf"`)
		assert.Contains(t, text, `search(query="<keyword>"`)
		assert.Contains(t, text, `target="qc-collection"`)

		// Ensure bibtex is not mentioned in the prompt
		assert.NotContains(t, strings.ToLower(text), "bibtex")
	})

	t.Run("success with topic only", func(t *testing.T) {
		res, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "review",
			Arguments: map[string]string{
				"topic": "deep reinforcement learning",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Messages, 1)

		textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)
		text := textContent.Text

		assert.Contains(t, text, "deep reinforcement learning")
		assert.NotContains(t, text, "**Target Collection**:")
		assert.Contains(t, text, `inquire(query="deep reinforcement learning", limit=300)`)
		assert.NotContains(t, strings.ToLower(text), "bibtex")
	})

	t.Run("missing topic returns error", func(t *testing.T) {
		_, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "review",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "topic argument is required")
	})

	t.Run("empty topic returns error", func(t *testing.T) {
		_, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "review",
			Arguments: map[string]string{
				"topic": "   ",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "topic argument is required")
	})
}

func TestGetPromptGaps(t *testing.T) {
	client := setupTestServer(t)

	t.Run("success with topic and target", func(t *testing.T) {
		res, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "gaps",
			Arguments: map[string]string{
				"topic":  "continual learning in robotics",
				"target": "robotics-collection",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Description, "continual learning in robotics")
		require.Len(t, res.Messages, 1)

		msg := res.Messages[0]
		assert.Equal(t, mcp.Role("user"), msg.Role)

		textContent, ok := msg.Content.(*mcp.TextContent)
		require.True(t, ok)
		text := textContent.Text

		assert.Contains(t, text, "continual learning in robotics")
		assert.Contains(t, text, "**Target Collection**: robotics-collection")
		assert.Contains(t, text, "Disconnected Trunk Lineages")
		assert.Contains(t, text, "Isolated or Ungrounded Leaves")
		assert.Contains(t, text, "Methodological Lag")
		assert.Contains(t, text, "Contradictory / Divergent Lineages")
		assert.Contains(t, text, "Network Sparsity via Graph Search")
		assert.Contains(t, text, `tos(category="root"`)
		assert.Contains(t, text, "not by whether it mentions the query")
		assert.Contains(t, text, `tos(category="trunk"`)
		assert.Contains(t, text, `tos(category="leaf"`)
		assert.Contains(t, text, `search(query="<concept>"`)
		assert.Contains(t, text, `target="robotics-collection"`)

		// Ensure bibtex is not mentioned in the prompt
		assert.NotContains(t, strings.ToLower(text), "bibtex")
	})

	t.Run("success with topic only", func(t *testing.T) {
		res, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "gaps",
			Arguments: map[string]string{
				"topic": "knowledge graphs",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Messages, 1)

		textContent, ok := res.Messages[0].Content.(*mcp.TextContent)
		require.True(t, ok)
		text := textContent.Text

		assert.Contains(t, text, "knowledge graphs")
		assert.NotContains(t, text, "**Target Collection**:")
		assert.Contains(t, text, `inquire(query="knowledge graphs", limit=300)`)
		assert.NotContains(t, strings.ToLower(text), "bibtex")
	})

	t.Run("missing topic returns error", func(t *testing.T) {
		_, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "gaps",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "topic argument is required")
	})

	t.Run("empty topic returns error", func(t *testing.T) {
		_, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
			Name: "gaps",
			Arguments: map[string]string{
				"topic": " \t ",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "topic argument is required")
	})
}

func TestUnknownPrompt(t *testing.T) {
	client := setupTestServer(t)

	_, err := client.GetPrompt(t.Context(), &mcp.GetPromptParams{
		Name: "unknown_prompt",
	})
	require.Error(t, err)
}
