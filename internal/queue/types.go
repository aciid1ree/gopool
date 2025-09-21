package queue

type State string

const (
	StateQueued  State = "queued"
	StateRunning State = "running"
	StateDone    State = "done"
	StateFailed  State = "failed"
)

type Task struct {
	ID         string
	Payload    string
	MaxRetries int
}

func (q *Q) Enqueue(task Task) error {
	if task.ID == "" {
		return ErrDuplicateId
	}

	q.mu.Lock()
	if !q.accepting {
		q.mu.Unlock()
		return ErrClosed
	}

	if q.store.Has(task.ID) {
		q.mu.Unlock()
		return ErrDuplicateId
	}

	select {
	case q.ch <- task:
		q.store.SetState(task.ID, StateQueued)
		q.mu.Unlock()
		return nil
	default:
		q.mu.Unlock()
		return ErrQueueFull
	}
}

func (q *Q) Take() <-chan Task {
	return q.ch
}

func (q *Q) Close() {
	q.mu.Lock()
	if !q.accepting {
		q.mu.Unlock()
		return
	}

	q.accepting = false
	close(q.ch)
	q.mu.Unlock()
}

func (q *Q) Capacity() int { return q.capacity }

func (q *Q) Len() int { return len(q.ch) }

func (q *Q) Accepting() bool {
	q.mu.Lock()
	ok := q.accepting
	q.mu.Unlock()
	return ok
}
