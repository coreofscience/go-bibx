package graphs_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/graphs"
	"github.com/hmdsefi/gograph"
	"github.com/stretchr/testify/assert"
)

func TestRemoveCycles(t *testing.T) {
	t.Run("DAG", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())
	})

	t.Run("graph with cycles", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")
		vD := gograph.NewVertex("D")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)
		_, _ = g.AddEdge(vC, vA) // Cycle A -> B -> C -> A
		_, _ = g.AddEdge(vD, vA)

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(1), newG.Order()) // A, B, C are removed, only D remains
		assert.Equal(t, uint32(0), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("D"))
	})

	t.Run("self loop", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vB) // Self loop on B

		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(1), newG.Order())
		assert.Equal(t, uint32(0), newG.Size())
		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.RemoveCycles(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("undirected graph error", func(t *testing.T) {
		g := gograph.New[string]()
		vA := gograph.NewVertex("A")
		g.AddVertex(vA)

		newG, err := graphs.RemoveCycles(g)
		assert.Error(t, err)
		assert.Nil(t, newG)
		assert.Contains(t, err.Error(), "must be directed")
	})
}

func TestRemoveDangling(t *testing.T) {
	t.Run("with dangling vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC) // C is a dangling vertex (outDegree=0, inDegree=1)

		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.Nil(t, newG.GetVertexByID("C"))
	})

	t.Run("without dangling vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)
		_, _ = g.AddEdge(vC, vA) // Cycle, no dangling vertices

		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(3), newG.Size())
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.RemoveDangling(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}

func TestInvert(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG := graphs.Invert(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")
		vC_new := newG.GetVertexByID("C")

		assert.NotNil(t, newG.GetEdge(vB_new, vA_new))
		assert.NotNil(t, newG.GetEdge(vC_new, vB_new))
		assert.Nil(t, newG.GetEdge(vA_new, vB_new))
		assert.Nil(t, newG.GetEdge(vB_new, vC_new))
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		newG := graphs.Invert(g)
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")

		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain
		assert.NotNil(t, newG.GetEdge(vB_new, vA_new))
		assert.Nil(t, newG.GetEdge(vA_new, vB_new))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG := graphs.Invert(g)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("undirected graph", func(t *testing.T) {
		g := gograph.New[string]()
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)

		newG := graphs.Invert(g)
		// It should just return the same graph pointer
		assert.Equal(t, g, newG)
	})
}

func TestKeep(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.Keep(g, []string{"A", "B"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(1), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.Nil(t, newG.GetVertexByID("C"))
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		newG, err := graphs.Keep(g, []string{"A", "C"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(0), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("C"))
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.Keep(g, []string{"A"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}

func TestPurge(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.Purge(g, []string{"B"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(0), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("C"))
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		newG, err := graphs.Purge(g, []string{"B"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(2), newG.Order())
		assert.Equal(t, uint32(0), newG.Size()) // Edge from A to B should be gone

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain
		assert.Nil(t, newG.GetVertexByID("B"))
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.Purge(g, []string{"A"})
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})
}

func TestUndirected(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		_, _ = g.AddEdge(vA, vB)
		_, _ = g.AddEdge(vB, vC)

		newG, err := graphs.Undirected(g)
		assert.NoError(t, err)
		assert.False(t, newG.IsDirected())
		assert.Equal(t, uint32(3), newG.Order())

		// Since it iterates over edges and might add A->B and B->A as separate operations (or similar)
		// gograph Undirected graph size count might be 4 if it double counts or just matches Size()
		assert.Equal(t, uint32(4), newG.Size())

		// Ensure edges exist both ways (or rather, no direction)
		vA_new := newG.GetVertexByID("A")
		vB_new := newG.GetVertexByID("B")
		vC_new := newG.GetVertexByID("C")

		assert.NotNil(t, newG.GetEdge(vA_new, vB_new))
		assert.NotNil(t, newG.GetEdge(vB_new, vA_new))
		assert.NotNil(t, newG.GetEdge(vB_new, vC_new))
		assert.NotNil(t, newG.GetEdge(vC_new, vB_new))
	})

	t.Run("isolated vertices", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		vC := gograph.NewVertex("C")

		g.AddVertex(vA)
		g.AddVertex(vB)
		g.AddVertex(vC)
		_, _ = g.AddEdge(vA, vB)

		newG, err := graphs.Undirected(g)
		assert.NoError(t, err)
		assert.False(t, newG.IsDirected())
		assert.Equal(t, uint32(3), newG.Order())
		assert.Equal(t, uint32(2), newG.Size())

		assert.NotNil(t, newG.GetVertexByID("A"))
		assert.NotNil(t, newG.GetVertexByID("B"))
		assert.NotNil(t, newG.GetVertexByID("C")) // Isolated vertex C should remain
	})

	t.Run("empty graph", func(t *testing.T) {
		g := gograph.New[string](gograph.Directed())
		newG, err := graphs.Undirected(g)
		assert.NoError(t, err)
		assert.Equal(t, uint32(0), newG.Order())
	})

	t.Run("already undirected", func(t *testing.T) {
		g := gograph.New[string]()
		vA := gograph.NewVertex("A")
		vB := gograph.NewVertex("B")
		_, _ = g.AddEdge(vA, vB)

		newG, err := graphs.Undirected(g)
		assert.NoError(t, err)
		// It should just return the same graph pointer
		assert.Equal(t, g, newG)
	})
}
