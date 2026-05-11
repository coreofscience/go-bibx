package collections_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestSet_Union(t *testing.T) {
	set1 := collections.NewSet[string]()
	set1.Add("a")
	set1.Add("b")

	set2 := collections.NewSet[string]()
	set2.Add("b")
	set2.Add("c")

	unionSet := set1.Union(set2)
	assert.NotNil(t, unionSet, "Union set should not be nil")
}

func TestSet_Add(t *testing.T) {
	set := collections.NewSet[string]()
	set.Add("a")
	set.Add("b")
	set.Add("c")

	assert.Equal(t, 3, set.Len(), "Set should contain 3 elements after adding 'a', 'b', and 'c'")

	// Adding duplicate elements should not change the length
	set.Add("a")
	assert.Equal(t, 3, set.Len(), "Set should still contain 3 elements after adding duplicate 'a'")
}

func TestSet_JSON(t *testing.T) {
	set := collections.NewSet[string]("a", "b", "c")

	data, err := set.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `["a","b","c"]`, string(data))

	var newSet collections.Set[string]
	err = newSet.UnmarshalJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, set.Len(), newSet.Len())
	assert.True(t, newSet.Contains("a"))
	assert.True(t, newSet.Contains("b"))
	assert.True(t, newSet.Contains("c"))
}
