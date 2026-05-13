package graphs

import (
	"fmt"
	"log/slog"

	"github.com/hmdsefi/gograph"
)

// RemoveDangling returns a new graph with no dangling vertices
func RemoveDangling[V comparable](g gograph.Graph[V]) (gograph.Graph[V], error) {
	dangling := make([]V, 0)
	for _, vertex := range g.GetAllVertices() {
		if vertex.OutDegree() == 0 && vertex.InDegree() == 1 {
			dangling = append(dangling, vertex.Label())
		}
	}
	slog.Debug("removing dangling vertices", "numDangling", len(dangling), "order", g.Order())
	newGraph, err := Purge(g, dangling)
	if err != nil {
		return nil, fmt.Errorf("failed to purge vertices: %w", err)
	}
	return newGraph, nil
}
