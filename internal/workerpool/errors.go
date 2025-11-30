package workerpool

import "errors"

var (
	ErrPoolNotStarted = errors.New("worker pool não foi iniciado")
	ErrPoolStopped    = errors.New("worker pool foi parado")
	ErrQueueFull      = errors.New("fila de tarefas está cheia")
)
