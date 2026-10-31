package theory

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const summary = `
bibx implements the Tree of Science (ToS) methodology for bibliographic data
research. It divides a bibliographic topic into 3 categories root, trunk and
leaves.

A "root" work is a seminal work, foundational to the given topic. These
works are identified mainly because lots of citation lineages end up with it
as an original resource.

Roots are expected to look a bit off topic. Topics seldom start under their
conventional modern names, so the works they grew from may use other terms or
come from neighbouring fields. Judge a root by whether the topic grew from it,
not by whether it mentions the query. Generic method citations, such as
reporting guidelines, are different: many papers in a collection may cite them,
but the topic did not grow from them.

A "trunk" work gives structure to the topic, it is where the topic became
the way it is today. These works are characterized by being join points
between two or more seminal works. These works are identified by having
lots of citation lineages crossing them.

A "leaf" work is a modern, state of the art, research paper about the given
topic. They might include meta-analysis papers, or just modern papers that cite
all the correct papers. These works are identified by having lots of outgoing
citation lineages that end up in the "root" papers.

Using the root, trunk and leaves you can build strong and useful literature
reviews. And identify which works need to be read to understand an specific
topic.
`

// Summary is a resource to hold the summary of the bibx algorithm.
type Summary struct{}

// NewSummary builds a [Summary] instance.
func NewSummary() *Summary {
	return &Summary{}
}

// Resource returns the [mcp.Resource] corresponding to this summary.
func (s *Summary) Resource() *mcp.Resource {
	return &mcp.Resource{
		Annotations: &mcp.Annotations{
			Audience: []mcp.Role{"user", "assistant"},
			Priority: 1,
		},
		Description: "Overview of the Tree of Science (ToS) bibliometric methodology (root, trunk, and leaf categories) used by bibx.",
		MIMEType:    "text/plain",
		Name:        "bibx summary",
		Size:        int64(len(summary)),
		Title:       "bibx and its relationship to the Tree of Science",
		URI:         "bibx:/summary",
	}
}

// Provide provides the resource back to the agent.
func (s *Summary) Provide(ctx context.Context, rrr *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	u, err := url.Parse(rrr.Params.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse provided resource URI: %w", err)
	}
	if u.Scheme != "bibx" {
		return nil, fmt.Errorf("URI scheme should be bibx, got: %q", u.Scheme)
	}
	return &mcp.ReadResourceResult{
		TTLMs:      int((24 * time.Hour).Milliseconds()),
		CacheScope: "public",
		Contents: []*mcp.ResourceContents{{
			URI:      "bibx:/summary",
			MIMEType: "text/plain",
			Text:     summary,
		}},
	}, nil
}

// Register registers the resource in the given [mcp.Server].
func (s *Summary) Register(server *mcp.Server) {
	server.AddResource(s.Resource(), s.Provide)
}
