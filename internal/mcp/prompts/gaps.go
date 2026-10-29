package prompts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	var targetHeader string
	var phase1Acquisition string
	var targetParam string

	if target != "" {
		targetHeader = fmt.Sprintf("\n**Target Collection**: %s", target)
		phase1Acquisition = fmt.Sprintf("- Verify the collection using `show(target=%q)` or review its overview at `bibx:/collections/%s/summary`.\n- If the collection needs to be initialized or refreshed, fetch literature from OpenAlex using `inquire(query=%q, limit=300, name=%q)`.", target, target, topic, target)
		targetParam = fmt.Sprintf(", target=%q", target)
	} else {
		phase1Acquisition = fmt.Sprintf("- Check existing collections with `list`. If no matching collection exists, fetch literature and construct the citation network using `inquire(query=%q, limit=300)`.", topic)
	}

	text := fmt.Sprintf(`You are an expert research scientist and bibliometrician tasked with identifying unaddressed research gaps, methodological blind spots, and open opportunities in:
**Research Field**: %s%s

Follow this systematic network-analysis methodology using the bibx MCP tools:

### Phase 1: Landscape Mapping & Knowledge Baseline
%s
- Understand the Tree of Science (ToS) model:
  Review `+"`bibx:/summary`"+` to understand the structural role of roots (foundations), trunks (methodological bridges), and leaves (modern frontiers).

### Phase 2: Mapping Established Ground vs Frontier
1. **Established Core (Roots & Trunks)**:
   - Query roots: `+"`tos(category=\"root\", limit=10%s)`"+` to establish what foundational problems and assumptions are already well-established.
   - Query trunks: `+"`tos(category=\"trunk\", limit=10%s)`"+` to identify dominant methodologies and standard analytical frameworks.
2. **Contemporary Frontier (Leaves)**:
   - Query leaves: `+"`tos(category=\"leaf\", limit=30%s)`"+` to survey active contemporary research papers.
   - Cluster leaves by sub-domain, application area, and theoretical orientation.

### Phase 3: Systematic Gap Detection Heuristics
Apply the following graph-topology heuristics to uncover research gaps:
1. **Disconnected Trunk Lineages (Interdisciplinary / Synthesis Gap)**:
   - Identify distinct trunk methodologies or root branches that co-exist in the field but share few or no citation bridges.
   - Gap question: Can methodology from Trunk A be combined with or applied to problems solved by Trunk B?
2. **Isolated or Ungrounded Leaves (Theoretical Grounding Gap)**:
   - Identify recent leaf papers exploring novel applications that have weak connections to established trunk frameworks.
   - Gap question: What foundational theory or formal rigor is missing from these empirical developments?
3. **Methodological Lag (Technique Transfer Gap)**:
   - Identify robust trunk methodologies that have successfully reshaped one branch of the field but have not yet been applied to active leaf clusters.
   - Gap question: Why has Method X not been applied to Sub-domain Y?
4. **Contradictory / Divergent Lineages (Consensus & Validation Gap)**:
   - Look for parallel trunk papers offering conflicting explanations, divergent empirical outcomes, or incompatible theoretical assumptions.
   - Gap question: What comparative experiments, benchmarking, or theoretical syntheses are needed to resolve this dispute?
5. **Network Sparsity via Graph Search**:
   - Test connectivity between pairs of concepts using `+"`search(query=\"<concept>\", limit=10%s)`"+`.
   - Look for sparse subgraphs, long path lengths, or absence of connecting bridge papers between complementary subtopics.

### Phase 4: Actionable Research Questions & Proposal Formulation
Translate the identified gaps into concrete, high-impact research proposals:
1. **Gap Characterization**: State the specific gap type and describe the missing link or discrepancy in the literature.
2. **Key Anchor Literature**: Reference the specific roots, trunks, or leaves involved.
3. **Proposed Research Question / Hypothesis**: Formulate clear, falsifiable hypotheses or engineering objectives addressing the gap.
4. **Methodological Approach**: Suggest an experimental or theoretical design capable of bridging the gap.`,
		topic,
		targetHeader,
		phase1Acquisition,
		targetParam,
		targetParam,
		targetParam,
		targetParam,
	)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Research gap analysis workflow for field: %s", topic),
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

// Register registers the gaps prompt on the provided MCP server.
func (p *Gaps) Register(server *mcp.Server) {
	server.AddPrompt(p.Prompt(), p.Handle)
}
