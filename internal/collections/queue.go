package collections

import "math/rand"

type UniqueQueue[T comparable] struct {
	queue   []T
	inQueue map[T]struct{}
}

// NewUniqueQueue creates a new UniqueQueue.
func NewUniqueQueue[T comparable](items ...T) *UniqueQueue[T] {
	q := &UniqueQueue[T]{
		queue:   make([]T, 0, len(items)),
		inQueue: make(map[T]struct{}),
	}
	for _, item := range items {
		q.Push(item)
	}
	return q
}

// Push adds an item to the queue if it is not already present.
func (q *UniqueQueue[T]) Push(item T) {
	if q == nil {
		return
	}
	if q.inQueue == nil {
		q.inQueue = make(map[T]struct{})
	}
	if _, ok := q.inQueue[item]; ok {
		return
	}
	q.queue = append(q.queue, item)
	q.inQueue[item] = struct{}{}
}

// Pop removes and returns the first item from the queue, if any.
func (q *UniqueQueue[T]) Pop() (T, bool) {
	var zero T
	if q == nil || len(q.queue) == 0 {
		return zero, false
	}
	item := q.queue[0]
	q.queue = q.queue[1:]
	if q.inQueue != nil {
		delete(q.inQueue, item)
	}
	return item, true
}

// Len returns the number of items in the queue.
func (q *UniqueQueue[T]) Len() int {
	if q == nil {
		return 0
	}
	return len(q.queue)
}

// Empty returns true if the queue is empty, false otherwise.
func (q *UniqueQueue[T]) Empty() bool {
	if q == nil {
		return true
	}
	return len(q.queue) == 0
}

// Shuffle shuffles the items in the queue.
func (q *UniqueQueue[T]) Shuffle() {
	if q == nil || len(q.queue) == 0 {
		return
	}
	rand.Shuffle(len(q.queue), func(i, j int) {
		q.queue[i], q.queue[j] = q.queue[j], q.queue[i]
	})
}
