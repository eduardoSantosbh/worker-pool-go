package workerpool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task representa uma tarefa a ser processada
type Task struct {
	ID      int
	Payload interface{}
	Handler func(interface{}) (interface{}, error)
	Result  chan Result
	Error   chan error
}

// Result representa o resultado do processamento
type Result struct {
	TaskID   int
	Payload  interface{}
	Output   interface{}
	Duration time.Duration
}

// WorkerPool gerencia um pool de workers
type WorkerPool struct {
	workerCount int
	taskQueue   chan Task
	workerPool  chan chan Task
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	mu          sync.RWMutex
	metrics     *Metrics
}

// Metrics armazena métricas do worker pool
type Metrics struct {
	TasksProcessed  int64
	TasksFailed     int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	mu              sync.RWMutex
}

// NewWorkerPool cria uma nova instância do WorkerPool
func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	if queueSize < 0 {
		queueSize = 0
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, queueSize),
		workerPool:  make(chan chan Task, workerCount),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &Metrics{},
	}
}

// Start inicia o worker pool
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return
	}

	// Inicia workers
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	// Inicia dispatcher
	wp.wg.Add(1)
	go wp.dispatcher()

	wp.started = true
}

// Stop para o worker pool
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.started {
		return
	}

	close(wp.taskQueue)
	wp.cancel()
	wp.wg.Wait()
	wp.started = false
}

// Submit adiciona uma tarefa ao pool
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if !wp.started {
		return ErrPoolNotStarted
	}

	select {
	case wp.taskQueue <- task:
		return nil
	case <-wp.ctx.Done():
		return ErrPoolStopped
	default:
		return ErrQueueFull
	}
}

// dispatcher distribui tarefas para workers disponíveis
func (wp *WorkerPool) dispatcher() {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			select {
			case workerTaskQueue := <-wp.workerPool:
				select {
				case workerTaskQueue <- task:
				case <-wp.ctx.Done():
					return
				}
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// worker processa tarefas
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	workerTaskQueue := make(chan Task)
	defer close(workerTaskQueue)

	// Log quando worker inicia
	fmt.Printf("  👷 Worker #%d iniciado e aguardando tarefas...\n", id)

	go func() {
		for {
			select {
			case wp.workerPool <- workerTaskQueue:
			case <-wp.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case task := <-workerTaskQueue:
			wp.processTask(task, id)

		case <-wp.ctx.Done():
			fmt.Printf("  🛑 Worker #%d finalizado\n", id)
			return
		}
	}
}

// processTask executa uma tarefa
func (wp *WorkerPool) processTask(task Task, workerID int) {
	startTime := time.Now()

	// Tenta extrair informação do payload para log mais detalhado
	payloadInfo := ""
	if task.Payload != nil {
		// Se o payload tiver método String() ou campos específicos, podemos usá-los
		if rec, ok := task.Payload.(interface{ GetName() string }); ok {
			payloadInfo = fmt.Sprintf(" - %s", rec.GetName())
		}
	}

	// Log quando worker recebe tarefa
	fmt.Printf("  [Worker #%d] ⚙️  Recebeu tarefa #%d%s\n", workerID, task.ID, payloadInfo)

	result, err := task.Handler(task.Payload)
	duration := time.Since(startTime)

	wp.updateMetrics(err, duration)

	if err != nil {
		fmt.Printf("  [Worker #%d] ❌ Tarefa #%d FALHOU após %v: %v\n", workerID, task.ID, duration, err)
		if task.Error != nil {
			task.Error <- err
		}
		return
	}

	fmt.Printf("  [Worker #%d] ✅ Tarefa #%d concluída em %v%s\n", workerID, task.ID, duration, payloadInfo)

	if task.Result != nil {
		task.Result <- Result{
			TaskID:   task.ID,
			Payload:  task.Payload,
			Output:   result,
			Duration: duration,
		}
	}
}

// updateMetrics atualiza as métricas
func (wp *WorkerPool) updateMetrics(err error, duration time.Duration) {
	wp.metrics.mu.Lock()
	defer wp.metrics.mu.Unlock()

	wp.metrics.TasksProcessed++
	wp.metrics.TotalDuration += duration
	if wp.metrics.TasksProcessed > 0 {
		wp.metrics.AverageDuration = wp.metrics.TotalDuration / time.Duration(wp.metrics.TasksProcessed)
	}

	if err != nil {
		wp.metrics.TasksFailed++
	}
}

// GetMetrics retorna as métricas atuais
func (wp *WorkerPool) GetMetrics() Metrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	return Metrics{
		TasksProcessed:  wp.metrics.TasksProcessed,
		TasksFailed:     wp.metrics.TasksFailed,
		TotalDuration:   wp.metrics.TotalDuration,
		AverageDuration: wp.metrics.AverageDuration,
	}
}

// GetWorkerCount retorna o número de workers
func (wp *WorkerPool) GetWorkerCount() int {
	return wp.workerCount
}

// IsRunning verifica se o pool está em execução
func (wp *WorkerPool) IsRunning() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.started
}
