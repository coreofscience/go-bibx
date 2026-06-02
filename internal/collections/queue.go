package collections

import (
	"errors"
	"math/rand/v2"
)

var ErrQueueEmpty = errors.New("queue is empty")

// UniqueQueue is a queue that only contains unique items.
type UniqueQueue[T comparable] struct {
	items   []T
	inQueue map[T]struct{}
}

// NewQueue creates a new UniqueQueue with the given items.
func NewQueue[T comparable](items ...T) *UniqueQueue[T] {
	uniqueItems := make([]T, 0, len(items))
	inQueue := make(map[T]struct{}, len(items))
	for _, item := range items {
		if _, ok := inQueue[item]; !ok {
			uniqueItems = append(uniqueItems, item)
			inQueue[item] = struct{}{}
		}
	}
	return &UniqueQueue[T]{
		items:   uniqueItems,
		inQueue: inQueue,
	}
}

// Enqueue adds an item to the queue if it is not already present.
func (q *UniqueQueue[T]) Enqueue(item T) bool {
	if _, ok := q.inQueue[item]; !ok {
		q.items = append(q.items, item)
		q.inQueue[item] = struct{}{}
		return true
	}
	return false
}

// Dequeue removes and returns the first item from the queue.
func (q *UniqueQueue[T]) Dequeue() (T, error) {
	if len(q.items) == 0 {
		var zero T
		return zero, ErrQueueEmpty
	}
	item := q.items[0]
	q.items = q.items[1:]
	delete(q.inQueue, item)
	return item, nil
}

// Empty returns true if the queue is empty.
func (q *UniqueQueue[T]) Empty() bool {
	return len(q.items) == 0
}

// Len returns the number of items in the queue.
func (q *UniqueQueue[T]) Len() int {
	return len(q.items)
}

// Shuffle shuffles the items in the queue.
func (q *UniqueQueue[T]) Shuffle() {
	rand.Shuffle(len(q.items), func(i, j int) {
		q.items[i], q.items[j] = q.items[j], q.items[i]
	})
}
