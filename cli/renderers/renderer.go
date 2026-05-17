package renderers

import (
	"io"

	"github.com/coreofscience/go-bibx/models"
)

type Renderer interface {
	// RenderResults renders the graph in a specific format.
	RenderResults(w io.Writer, result []*models.Result) error

	// RenderArticle renders a single article in a specific format.
	RenderArticle(w io.Writer, article *models.Article) error

	// RenderAnalysis renders an analysis in a specific format.
	RenderAnalysis(w io.Writer, analysis *models.Analysis) error
}
