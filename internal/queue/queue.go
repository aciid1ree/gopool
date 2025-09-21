package queue

import (
	"sync"
)

type Q struct {
	ch        chan Task
	capacity  int
	mu        sync.Mutex
	accepting bool
	store     Store
}

func New(capacity int, store Store) *Q {
	if capacity <= 0 {
		capacity = 1
	}
	return &Q{
		ch:        make(chan Task, capacity),
		capacity:  capacity,
		store:     store,
		accepting: true,
	}
}
