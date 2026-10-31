package prompts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	var targetHeader string
	var phase1Acquisition string
	var targetParam string

	if target != "" {
		targetHeader = fmt.Sprintf("\n**Target Collection**: %s", target)
		phase1Acquisition = fmt.Sprintf("- Verify the collection using `show(target=%q)` or inspect its summary at `bibx:/collections/%s/summary`.\n- If the collection needs to be refreshed or does not exist, fetch literature from OpenAlex using `inquire(query=%q, limit=300, name=%q)`.", target, target, topic, target)
		targetParam = fmt.Sprintf(", target=%q", target)
	} else {
		phase1Acquisition = fmt.Sprintf("- Check existing collections using `list`. If no relevant collection exists, fetch and analyze literature from OpenAlex by calling `inquire(query=%q, limit=300)`.", topic)
	}

	text := fmt.Sprintf(`You are an expert academic researcher and bibliometrician conducting a systematic literature review on:
**Topic**: %s%s

Follow this step-by-step methodology using the bibx MCP tools:

### Phase 1: Collection Acquisition & Domain Scope
%s
- Understand the Tree of Science (ToS) methodology:
  Read resource `+"`bibx:/summary`"+` to understand how literature is categorized into foundational origins (Roots), structural methodologies (Trunks), and contemporary frontiers (Leaves).

### Phase 2: Bibliometric Triangulation via Tree of Science
Systematically explore the field across its evolutionary dimensions using the `+"`tos`"+` tool:
1. **Foundational Heritage (Roots)**:
   - Call `+"`tos(category=\"root\", limit=10%s)`"+`
   - Identify seminal papers, original theoretical frameworks, founding authors, and initial problem formulations.
   - Expect roots to look a bit off topic, because topics seldom start under their modern names. Judge a root by whether the topic grew from it, not by whether it mentions the query, and set aside generic method citations such as reporting guidelines.
   - Note key baseline assumptions and conceptual foundations.
2. **Structural Evolution (Trunks)**:
   - Call `+"`tos(category=\"trunk\", limit=10%s)`"+`
   - Identify major methodology papers, meta-syntheses, and bridge papers connecting different seminal branches.
   - Trace how the field evolved, diversified, and formed dominant schools of thought.
3. **Current State of the Art (Leaves)**:
   - Call `+"`tos(category=\"leaf\", limit=20%s)`"+`
   - Identify recent frontier works, emerging subtopics, modern experimental validation, and specialized applications.
   - Group leaves by thematic focus and technological approaches.
4. **Deep-Dive Abstract Inspection**:
   - Because `+"`tos`"+` and `+"`search`"+` omit abstracts by default to save context window, call `+"`works`"+` with the IDs of the most important papers to read their full abstracts and metadata.

### Phase 3: Thematic Drill-Down via Graph Search
- For key subtopics, conflicting paradigms, or specialized questions that emerge from your analysis of leaves and trunks, use the `+"`search`"+` tool:
  - Call `+"`search(query=\"<keyword>\", limit=10%s)`"+`
  - Analyze the Steiner tree subgraph returned: examine connecting bridge nodes and citation paths between seed works and the broader network.

### Phase 4: Literature Review Synthesis & Report Structure
Synthesize your findings into a rigorous, publishable literature review structured as follows:
1. **Introduction & Scope**: Define the research domain, scope boundaries, and bibliometric landscape overview.
2. **Foundational Roots**: Critical examination of seminal papers and core principles upon which the field rests.
3. **Structural Methodologies & Evolution (Trunks)**: The evolutionary trajectory of methods, paradigms, and transitional breakthroughs.
4. **Contemporary Frontiers (Leaves)**: State of the art, competing approaches, and active empirical frontiers.
5. **Critical Discussion & Future Perspectives**: Synthesis of methodological consensus vs debate, unanswered questions, and emerging directions.`,
		topic,
		targetHeader,
		phase1Acquisition,
		targetParam,
		targetParam,
		targetParam,
		targetParam,
	)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Systematic literature review workflow for topic: %s", topic),
		Messages: []*mcp.PromptMessage{
			{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: text,
				},
			},
		},
	}, nil
}

// Register registers the review prompt on the provided MCP server.
func (p *Review) Register(server *mcp.Server) {
	server.AddPrompt(p.Prompt(), p.Handle)
}
