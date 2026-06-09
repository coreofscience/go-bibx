package collections_test

import (
	"testing"

	"github.com/coreofscience/go-bibx/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestNewUniqueQueue(t *testing.T) {
	q := collections.NewUniqueQueue(1, 2, 2, 3)
	assert.NotNil(t, q)
	assert.Equal(t, 3, q.Len())
	assert.False(t, q.Empty())

	val, ok := q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 2, val)

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 3, val)

	val, ok = q.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestUniqueQueue_PushAndPop(t *testing.T) {
	q := collections.NewUniqueQueue[string]()
	assert.True(t, q.Empty())
	assert.Equal(t, 0, q.Len())

	q.Push("a")
	assert.False(t, q.Empty())
	assert.Equal(t, 1, q.Len())

	// Push duplicate
	q.Push("a")
	assert.Equal(t, 1, q.Len())

	q.Push("b")
	assert.Equal(t, 2, q.Len())

	// Pop first item
	val, ok := q.Pop()
	assert.True(t, ok)
	assert.Equal(t, "a", val)
	assert.Equal(t, 1, q.Len())

	// Push "a" again (it was popped, so it should be allowed again)
	q.Push("a")
	assert.Equal(t, 2, q.Len())

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, "b", val)

	val, ok = q.Pop()
	assert.True(t, ok)
	assert.Equal(t, "a", val)

	_, ok = q.Pop()
	assert.False(t, ok)
}

func TestUniqueQueue_LazyInitialization(t *testing.T) {
	// A queue initialized with empty struct
	q := &collections.UniqueQueue[int]{}
	assert.Equal(t, 0, q.Len())
	assert.True(t, q.Empty())

	// This shouldn't panic
	q.Push(10)
	assert.Equal(t, 1, q.Len())
	assert.False(t, q.Empty())

	val, ok := q.Pop()
	assert.True(t, ok)
	assert.Equal(t, 10, val)
}

func TestUniqueQueue_NilReceiver(t *testing.T) {
	var q *collections.UniqueQueue[int]

	assert.Equal(t, 0, q.Len())
	assert.True(t, q.Empty())

	// Verify operations do not panic
	q.Push(5)
	val, ok := q.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	assert.NotPanics(t, func() {
		q.Shuffle()
	})
}

func TestUniqueQueue_Shuffle(t *testing.T) {
	q := collections.NewUniqueQueue[int]()
	// Shuffle empty
	assert.NotPanics(t, func() {
		q.Shuffle()
	})

	// Shuffle with elements
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, item := range items {
		q.Push(item)
	}

	assert.Equal(t, 10, q.Len())
	q.Shuffle()
	assert.Equal(t, 10, q.Len())

	// Retrieve all elements and verify we got exactly the same set of elements
	poppedMap := make(map[int]bool)
	for q.Len() > 0 {
		val, ok := q.Pop()
		assert.True(t, ok)
		poppedMap[val] = true
	}

	assert.Equal(t, len(items), len(poppedMap))
	for _, item := range items {
		assert.True(t, poppedMap[item], "Expected item %d to be in the popped elements", item)
	}
}
