package union_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/union"
	"github.com/stretchr/testify/assert"
)

func TestUnionFind_New(t *testing.T) {
	items := []string{"a", "b", "c"}
	uf := union.New(items)
	assert.NotNil(t, uf)

	for _, item := range items {
		assert.Equal(t, item, uf.Find(item))
	}
}

func TestUnionFind_Union(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	uf := union.New(items)

	uf.Union("a", "b")
	assert.True(t, uf.Connected("a", "b"))
	assert.Equal(t, uf.Find("a"), uf.Find("b"))

	uf.Union("c", "d")
	assert.True(t, uf.Connected("c", "d"))
	assert.False(t, uf.Connected("a", "c"))

	uf.Union("b", "c")
	assert.True(t, uf.Connected("a", "d"))
	assert.True(t, uf.Connected("b", "d"))
}

func TestUnionFind_Union_SameSet(t *testing.T) {
	items := []string{"a", "b"}
	uf := union.New(items)
	uf.Union("a", "b")

	rootBefore := uf.Find("a")
	uf.Union("a", "b")
	rootAfter := uf.Find("a")

	assert.Equal(t, rootBefore, rootAfter)
}

func TestUnionFind_Find_NonExistent(t *testing.T) {
	uf := union.New([]string{"a"})
	assert.Equal(t, "b", uf.Find("b"))
}

func TestUnionFind_Connected(t *testing.T) {
	items := []string{"a", "b", "c"}
	uf := union.New(items)

	assert.False(t, uf.Connected("a", "b"))
	uf.Union("a", "b")
	assert.True(t, uf.Connected("a", "b"))
	assert.False(t, uf.Connected("a", "c"))
}

func TestUnionFind_Components(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	uf := union.New(items)

	uf.Union("a", "b")
	uf.Union("c", "d")

	components := uf.Components()
	assert.Len(t, components, 3)

	var foundAB, foundCD, foundE bool
	for _, comp := range components {
		if len(comp) == 2 {
			if (comp[0] == "a" && comp[1] == "b") || (comp[0] == "b" && comp[1] == "a") {
				foundAB = true
			} else if (comp[0] == "c" && comp[1] == "d") || (comp[0] == "d" && comp[1] == "c") {
				foundCD = true
			}
		} else if len(comp) == 1 && comp[0] == "e" {
			foundE = true
		}
	}

	assert.True(t, foundAB, "Component {a, b} not found")
	assert.True(t, foundCD, "Component {c, d} not found")
	assert.True(t, foundE, "Component {e} not found")
}
