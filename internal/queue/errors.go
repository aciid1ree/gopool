package queue

import "errors"

var (
	// ErrDuplicateId ErrDuplicateID возвращается, если задача с таким же идентификатором уже существует.
	ErrDuplicateId = errors.New("duplicate task id")

	// ErrQueueFull указывает на то, что буфер уже заполнен
	ErrQueueFull = errors.New("queue is full")

	// ErrClosed указывает, что очередь закрыта и не будет принимать новые задачи.
	ErrClosed = errors.New("queue is closed")
)
