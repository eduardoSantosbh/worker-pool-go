package workerpool

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		queueSize   int
		expected    int
	}{
		{"Normal pool", 5, 10, 5},
		{"Zero workers", 0, 10, 1},
		{"Negative workers", -5, 10, 1},
		{"Negative queue size", 5, -10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewWorkerPool(tt.workerCount, tt.queueSize)
			if pool.GetWorkerCount() != tt.expected {
				t.Errorf("Expected %d workers, got %d", tt.expected, pool.GetWorkerCount())
			}
		})
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool := NewWorkerPool(3, 10)

	if pool.IsRunning() {
		t.Error("Pool should not be running before Start()")
	}

	pool.Start()
	if !pool.IsRunning() {
		t.Error("Pool should be running after Start()")
	}

	// Aguarda um pouco para workers iniciarem
	time.Sleep(50 * time.Millisecond)

	pool.Stop()
	if pool.IsRunning() {
		t.Error("Pool should not be running after Stop()")
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	pool.Start()
	defer pool.Stop()

	done := make(chan bool, 1)

	task := Task{
		ID:      1,
		Payload: "test",
		Handler: func(payload interface{}) (interface{}, error) {
			return "result", nil
		},
		Result: make(chan Result, 1),
	}

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	select {
	case result := <-task.Result:
		if result.TaskID != 1 {
			t.Errorf("Expected TaskID 1, got %d", result.TaskID)
		}
		if result.Output != "result" {
			t.Errorf("Expected output 'result', got %v", result.Output)
		}
		done <- true
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestWorkerPool_SubmitBeforeStart(t *testing.T) {
	pool := NewWorkerPool(2, 10)

	task := Task{
		ID:      1,
		Payload: "test",
		Handler: func(payload interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	err := pool.Submit(task)
	if err != ErrPoolNotStarted {
		t.Errorf("Expected ErrPoolNotStarted, got %v", err)
	}
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	pool.Start()
	defer pool.Stop()

	errChan := make(chan error, 1)

	task := Task{
		ID:      1,
		Payload: "test",
		Handler: func(payload interface{}) (interface{}, error) {
			return nil, errors.New("processing error")
		},
		Error: errChan,
	}

	pool.Submit(task)

	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "processing error" {
			t.Errorf("Expected 'processing error', got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for error")
	}
}

func TestWorkerPool_Metrics(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	pool.Start()
	defer pool.Stop()

	// Submete algumas tarefas
	for i := 0; i < 5; i++ {
		task := Task{
			ID:      i,
			Payload: i,
			Handler: func(payload interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return payload, nil
			},
		}
		pool.Submit(task)
	}

	// Aguarda processamento
	time.Sleep(200 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.TasksProcessed == 0 {
		t.Error("Expected tasks to be processed")
	}
}

func TestWorkerPool_ConcurrentSubmits(t *testing.T) {
	pool := NewWorkerPool(10, 100)
	pool.Start()
	defer pool.Stop()

	results := make(chan Result, 50)
	var wg sync.WaitGroup

	// Submete 50 tarefas concorrentemente
	for i := 0; i < 50; i++ {
		wg.Add(1)
		taskID := i
		task := Task{
			ID:      taskID,
			Payload: taskID,
			Handler: func(payload interface{}) (interface{}, error) {
				return payload, nil
			},
			Result: results,
		}

		go func(tsk Task) {
			defer wg.Done()
			if err := pool.Submit(tsk); err != nil {
				t.Errorf("Error submitting task: %v", err)
			}
		}(task)
	}

	// Aguarda todas as submissões
	wg.Wait()

	// Aguarda processamento
	time.Sleep(500 * time.Millisecond)

	// Coleta resultados
	received := 0
	timeout := time.After(5 * time.Second)

	for received < 50 {
		select {
		case <-results:
			received++
		case <-timeout:
			t.Fatalf("Timeout: received %d of 50 results", received)
		}
	}

	if received != 50 {
		t.Errorf("Expected 50 results, got %d", received)
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	pool := NewWorkerPool(1, 2) // Queue muito pequena
	pool.Start()
	defer pool.Stop()

	// Preenche a fila
	for i := 0; i < 2; i++ {
		task := Task{
			ID:      i,
			Payload: i,
			Handler: func(payload interface{}) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			},
		}
		pool.Submit(task)
	}

	// Tenta submeter com fila cheia
	task := Task{
		ID:      99,
		Payload: 99,
		Handler: func(payload interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	err := pool.Submit(task)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

func TestWorkerPool_MultipleStarts(t *testing.T) {
	pool := NewWorkerPool(3, 10)

	pool.Start()
	pool.Start() // Chamar Start() duas vezes não deve criar workers extras
	pool.Start()

	time.Sleep(50 * time.Millisecond)

	// Verifica se ainda está rodando
	if !pool.IsRunning() {
		t.Error("Pool should still be running")
	}

	pool.Stop()
}

func TestWorkerPool_StopWithoutStart(t *testing.T) {
	pool := NewWorkerPool(3, 10)

	// Deve ser seguro chamar Stop() sem Start()
	pool.Stop()

	if pool.IsRunning() {
		t.Error("Pool should not be running")
	}
}

func TestWorkerPool_MetricsAfterStop(t *testing.T) {
	pool := NewWorkerPool(2, 10)
	pool.Start()

	// Processa algumas tarefas
	for i := 0; i < 3; i++ {
		task := Task{
			ID:      i,
			Payload: i,
			Handler: func(payload interface{}) (interface{}, error) {
				return payload, nil
			},
		}
		pool.Submit(task)
	}

	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	// Métricas devem estar disponíveis após stop
	metrics := pool.GetMetrics()
	if metrics.TasksProcessed == 0 {
		t.Error("Expected metrics to be available after stop")
	}
}

