package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/csvreader"
	"github.com/seu-usuario/worker-pool-csv-processor/internal/database"
	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
	"github.com/seu-usuario/worker-pool-csv-processor/internal/validator"
	"github.com/seu-usuario/worker-pool-csv-processor/internal/workerpool"
)

func main() {
	// Parse de flags de linha de comando
	var (
		csvFile   = flag.String("csv", "data/employees.csv", "Caminho do arquivo CSV")
		dbPath    = flag.String("db", "employees.db", "Caminho do banco de dados SQLite")
		workers   = flag.Int("workers", runtime.NumCPU()*2, "Número de workers")
		queueSize = flag.Int("queue", 100, "Tamanho da fila de tarefas")
		showStats = flag.Bool("stats", false, "Mostra estatísticas do banco e sai")
	)
	flag.Parse()

	// Se apenas quer ver stats
	if *showStats {
		showDatabaseStats(*dbPath)
		return
	}

	// Valida arquivo CSV
	if _, err := os.Stat(*csvFile); os.IsNotExist(err) {
		log.Fatalf("❌ Arquivo CSV não encontrado: %s", *csvFile)
	}

	fmt.Println("🚀 Worker Pool CSV Processor")
	fmt.Println("============================")
	fmt.Printf("📄 Arquivo CSV: %s\n", *csvFile)
	fmt.Printf("💾 Banco de dados: %s\n", *dbPath)
	fmt.Printf("👷 Workers: %d\n", *workers)
	fmt.Printf("📋 Tamanho da fila: %d\n\n", *queueSize)

	// Inicia processamento
	processCSV(*csvFile, *dbPath, *workers, *queueSize)
}

func processCSV(csvFile, dbPath string, workerCount, queueSize int) {
	startTime := time.Now()

	// 1. Abre conexão com banco de dados
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("❌ Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// 2. Lê arquivo CSV
	fmt.Println("📖 Lendo arquivo CSV...")
	csvReader := csvreader.NewReader(csvFile)
	records, parseErrors, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("❌ Erro ao ler CSV: %v", err)
	}

	fmt.Printf("✅ %d registros lidos do CSV\n", len(records))
	if len(parseErrors) > 0 {
		fmt.Printf("⚠️  %d erros ao parsear linhas:\n", len(parseErrors))
		for _, e := range parseErrors[:min(5, len(parseErrors))] {
			fmt.Printf("   - %v\n", e)
		}
		if len(parseErrors) > 5 {
			fmt.Printf("   ... e mais %d erros\n", len(parseErrors)-5)
		}
	}
	fmt.Println()

	// 3. Cria validador
	validator := validator.NewValidator()

	// 4. Cria Worker Pool
	fmt.Printf("🏭 Criando Worker Pool com %d workers...\n", workerCount)
	pool := workerpool.NewWorkerPool(workerCount, queueSize)
	fmt.Printf("🚀 Iniciando workers...\n\n")
	pool.Start()
	defer pool.Stop()

	// Aguarda um momento para workers iniciarem
	time.Sleep(100 * time.Millisecond)

	// 5. Processa registros
	var wg sync.WaitGroup
	var mu sync.Mutex
	var (
		processedCount int
		successCount   int
		failedCount    int
		results        []models.ProcessingResult
	)

	// Canal para coletar resultados
	resultsChan := make(chan models.ProcessingResult, len(records))

	// Submete tarefas ao pool
	fmt.Printf("📤 Submetendo %d tarefas ao Worker Pool...\n\n", len(records))
	for i, record := range records {
		recordCopy := record // Importante: cópia para closure

		task := workerpool.Task{
			ID:      i + 1,
			Payload: recordCopy,
			Handler: func(payload interface{}) (interface{}, error) {
				rec := payload.(*models.Record)

				// Valida registro
				if err := validator.Validate(rec); err != nil {
					return models.ProcessingResult{
						RowNumber: rec.RowNumber,
						Record:    rec,
						Success:   false,
						Error:     err,
					}, nil
				}

				// Insere no banco de dados
				if err := db.InsertRecord(rec); err != nil {
					return models.ProcessingResult{
						RowNumber: rec.RowNumber,
						Record:    rec,
						Success:   false,
						Error:     err,
					}, nil
				}

				return models.ProcessingResult{
					RowNumber: rec.RowNumber,
					Record:    rec,
					Success:   true,
				}, nil
			},
			Result: make(chan workerpool.Result, 1),
			Error:  make(chan error, 1),
		}

		if err := pool.Submit(task); err != nil {
			fmt.Printf("  ❌ Erro ao submeter tarefa %d: %v\n", i+1, err)
			continue
		}

		// Coleta resultado
		wg.Add(1)
		go func(t workerpool.Task) {
			defer wg.Done()
			select {
			case result := <-t.Result:
				if pr, ok := result.Output.(models.ProcessingResult); ok {
					pr.Duration = result.Duration
					resultsChan <- pr
				}
			case err := <-t.Error:
				fmt.Printf("  ❌ Erro ao processar tarefa %d: %v\n", t.ID, err)
			case <-time.After(30 * time.Second):
				fmt.Printf("⏱️  Timeout processando tarefa %d\n", t.ID)
			}
		}(task)
	}

	// Aguarda todos os resultados
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Processa resultados
	fmt.Println("\n⏳ Aguardando processamento...")
	fmt.Println()
	progressTicker := time.NewTicker(500 * time.Millisecond)
	defer progressTicker.Stop()

	done := make(chan bool)
	go func() {
		for result := range resultsChan {
			mu.Lock()
			processedCount++
			if result.Success {
				successCount++
			} else {
				failedCount++
			}
			results = append(results, result)
			mu.Unlock()
		}
		done <- true
	}()

	// Mostra progresso periódico
	go func() {
		for {
			select {
			case <-progressTicker.C:
				mu.Lock()
				currentProcessed := processedCount
				currentSuccess := successCount
				currentFailed := failedCount
				mu.Unlock()

				if currentProcessed < len(records) {
					fmt.Printf("  📊 Progresso: %d/%d processados (✓ %d, ✗ %d)\n",
						currentProcessed, len(records), currentSuccess, currentFailed)
				}
			case <-done:
				return
			}
		}
	}()

	<-done
	fmt.Println() // Nova linha após progresso

	// 6. Mostra estatísticas finais
	totalDuration := time.Since(startTime)
	poolMetrics := pool.GetMetrics()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 RESULTADOS DO PROCESSAMENTO")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("✅ Sucesso: %d registros\n", successCount)
	fmt.Printf("❌ Falhas: %d registros\n", failedCount)
	fmt.Printf("📝 Total processado: %d registros\n", processedCount)
	fmt.Printf("⏱️  Tempo total: %v\n", totalDuration)
	fmt.Printf("⚡ Throughput: %.2f registros/segundo\n\n", float64(processedCount)/totalDuration.Seconds())

	fmt.Println("📈 MÉTRICAS DO WORKER POOL")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Tarefas processadas: %d\n", poolMetrics.TasksProcessed)
	fmt.Printf("Tarefas falharam: %d\n", poolMetrics.TasksFailed)
	fmt.Printf("Duração média: %v\n", poolMetrics.AverageDuration)

	// 7. Mostra alguns erros (se houver)
	if failedCount > 0 {
		fmt.Println("\n⚠️  PRIMEIROS ERROS ENCONTRADOS:")
		fmt.Println(strings.Repeat("-", 50))
		errorCount := 0
		for _, result := range results {
			if !result.Success && errorCount < 5 {
				fmt.Printf("Linha %d: %v\n", result.RowNumber, result.Error)
				errorCount++
			}
		}
		if failedCount > 5 {
			fmt.Printf("... e mais %d erros\n", failedCount-5)
		}
	}

	// 8. Estatísticas do banco de dados
	fmt.Println("\n💾 ESTATÍSTICAS DO BANCO DE DADOS")
	fmt.Println(strings.Repeat("-", 50))
	stats, err := db.GetStats()
	if err != nil {
		log.Printf("❌ Erro ao obter estatísticas: %v", err)
	} else {
		fmt.Printf("Total de registros: %d\n", stats["total"])
		fmt.Printf("Ativos: %d\n", stats["active"])
		fmt.Printf("Inativos: %d\n", stats["inactive"])
		if byDept, ok := stats["by_department"].(map[string]int); ok {
			fmt.Println("\nPor departamento:")
			for dept, count := range byDept {
				fmt.Printf("  %s: %d\n", dept, count)
			}
		}
	}

	fmt.Println("\n✅ Processamento concluído!")
}

func showDatabaseStats(dbPath string) {
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("❌ Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		log.Fatalf("❌ Erro ao obter estatísticas: %v", err)
	}

	fmt.Println("📊 ESTATÍSTICAS DO BANCO DE DADOS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total de registros: %d\n", stats["total"])
	fmt.Printf("Ativos: %d\n", stats["active"])
	fmt.Printf("Inativos: %d\n", stats["inactive"])

	if byDept, ok := stats["by_department"].(map[string]int); ok {
		fmt.Println("\nPor departamento:")
		for dept, count := range byDept {
			fmt.Printf("  %s: %d\n", dept, count)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
