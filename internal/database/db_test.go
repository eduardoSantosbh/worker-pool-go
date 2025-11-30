package database

import (
	"os"
	"testing"
	"time"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
)

func createTestDB(t *testing.T) (*DB, string) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	db, err := NewDB(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	return db, tmpfile.Name()
}

func TestNewDB(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	if db == nil {
		t.Fatal("Expected database instance, got nil")
	}
}

func TestCreateTables(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	// Se chegou aqui, as tabelas foram criadas
	// Verifica se pode inserir um registro
	record := &models.Record{
		Name:        "Test User",
		Email:       "test@example.com",
		Age:         30,
		Salary:      5000.00,
		Department:  "TI",
		IsActive:    true,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		RowNumber:   1,
	}

	err := db.InsertRecord(record)
	if err != nil {
		t.Fatalf("Expected no error inserting record, got %v", err)
	}
}

func TestInsertRecord(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	record := &models.Record{
		Name:        "João Silva",
		Email:       "joao@empresa.com",
		Age:         28,
		Salary:      5500.00,
		Department:  "TI",
		IsActive:    true,
		CreatedAt:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		ProcessedAt: time.Now(),
		RowNumber:   1,
	}

	err := db.InsertRecord(record)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestInsertRecord_DuplicateEmail(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	record1 := &models.Record{
		Name:        "João Silva",
		Email:       "joao@empresa.com",
		Age:         28,
		Salary:      5500.00,
		Department:  "TI",
		IsActive:    true,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		RowNumber:   1,
	}

	record2 := &models.Record{
		Name:        "João Santos",
		Email:       "joao@empresa.com", // Mesmo email
		Age:         30,
		Salary:      6000.00,
		Department:  "RH",
		IsActive:    true,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		RowNumber:   2,
	}

	// Insere primeiro registro
	err := db.InsertRecord(record1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Insere segundo registro com mesmo email (deve fazer upsert)
	err = db.InsertRecord(record2)
	if err != nil {
		t.Fatalf("Expected no error on duplicate email (upsert), got %v", err)
	}

	// Verifica se o registro foi atualizado
	retrieved, err := db.GetRecordByEmail("joao@empresa.com")
	if err != nil {
		t.Fatalf("Expected no error retrieving record, got %v", err)
	}

	// Deve ter os dados do segundo registro (última inserção)
	if retrieved.Name != "João Santos" {
		t.Errorf("Expected name 'João Santos' (after upsert), got '%s'", retrieved.Name)
	}
	if retrieved.Department != "RH" {
		t.Errorf("Expected department 'RH' (after upsert), got '%s'", retrieved.Department)
	}
}

func TestInsertRecord_MultipleRecords(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	records := []*models.Record{
		{
			Name:        "João Silva",
			Email:       "joao@empresa.com",
			Age:         28,
			Salary:      5500.00,
			Department:  "TI",
			IsActive:    true,
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
			RowNumber:   1,
		},
		{
			Name:        "Maria Santos",
			Email:       "maria@empresa.com",
			Age:         32,
			Salary:      6200.00,
			Department:  "RH",
			IsActive:    true,
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
			RowNumber:   2,
		},
		{
			Name:        "Pedro Oliveira",
			Email:       "pedro@empresa.com",
			Age:         45,
			Salary:      8500.00,
			Department:  "Financeiro",
			IsActive:    false,
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
			RowNumber:   3,
		},
	}

	for _, record := range records {
		err := db.InsertRecord(record)
		if err != nil {
			t.Fatalf("Expected no error inserting record, got %v", err)
		}
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Expected no error getting stats, got %v", err)
	}

	if stats["total"].(int) != 3 {
		t.Errorf("Expected 3 total records, got %d", stats["total"])
	}
}

func TestGetStats(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	// Insere alguns registros
	records := []*models.Record{
		{Name: "João", Email: "joao@test.com", Age: 28, Salary: 5000, Department: "TI", IsActive: true, CreatedAt: time.Now(), ProcessedAt: time.Now(), RowNumber: 1},
		{Name: "Maria", Email: "maria@test.com", Age: 32, Salary: 6000, Department: "RH", IsActive: true, CreatedAt: time.Now(), ProcessedAt: time.Now(), RowNumber: 2},
		{Name: "Pedro", Email: "pedro@test.com", Age: 45, Salary: 7000, Department: "TI", IsActive: false, CreatedAt: time.Now(), ProcessedAt: time.Now(), RowNumber: 3},
	}

	for _, record := range records {
		db.InsertRecord(record)
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats["total"].(int) != 3 {
		t.Errorf("Expected total 3, got %d", stats["total"])
	}

	if stats["active"].(int) != 2 {
		t.Errorf("Expected active 2, got %d", stats["active"])
	}

	if stats["inactive"].(int) != 1 {
		t.Errorf("Expected inactive 1, got %d", stats["inactive"])
	}

	byDept, ok := stats["by_department"].(map[string]int)
	if !ok {
		t.Fatal("Expected by_department to be map[string]int")
	}

	if byDept["TI"] != 2 {
		t.Errorf("Expected TI department count 2, got %d", byDept["TI"])
	}

	if byDept["RH"] != 1 {
		t.Errorf("Expected RH department count 1, got %d", byDept["RH"])
	}
}

func TestGetRecordByEmail(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	record := &models.Record{
		Name:        "João Silva",
		Email:       "joao@empresa.com",
		Age:         28,
		Salary:      5500.00,
		Department:  "TI",
		IsActive:    true,
		CreatedAt:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		ProcessedAt: time.Now(),
		RowNumber:   1,
	}

	err := db.InsertRecord(record)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	retrieved, err := db.GetRecordByEmail("joao@empresa.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if retrieved.Name != "João Silva" {
		t.Errorf("Expected name 'João Silva', got '%s'", retrieved.Name)
	}

	if retrieved.Email != "joao@empresa.com" {
		t.Errorf("Expected email 'joao@empresa.com', got '%s'", retrieved.Email)
	}

	if retrieved.Age != 28 {
		t.Errorf("Expected age 28, got %d", retrieved.Age)
	}
}

func TestGetRecordByEmail_NotFound(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	_, err := db.GetRecordByEmail("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for nonexistent email, got nil")
	}
}

func TestCleanup(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	// Insere alguns registros
	record := &models.Record{
		Name:        "Test User",
		Email:       "test@example.com",
		Age:         30,
		Salary:      5000.00,
		Department:  "TI",
		IsActive:    true,
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
		RowNumber:   1,
	}

	db.InsertRecord(record)

	// Verifica que tem registro
	stats, _ := db.GetStats()
	if stats["total"].(int) != 1 {
		t.Fatalf("Expected 1 record before cleanup, got %d", stats["total"])
	}

	// Limpa
	err := db.Cleanup()
	if err != nil {
		t.Fatalf("Expected no error on cleanup, got %v", err)
	}

	// Verifica que não tem mais registros
	stats, _ = db.GetStats()
	if stats["total"].(int) != 0 {
		t.Errorf("Expected 0 records after cleanup, got %d", stats["total"])
	}
}

func TestGetStats_EmptyDatabase(t *testing.T) {
	db, filePath := createTestDB(t)
	defer os.Remove(filePath)
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats["total"].(int) != 0 {
		t.Errorf("Expected total 0, got %d", stats["total"])
	}

	if stats["active"].(int) != 0 {
		t.Errorf("Expected active 0, got %d", stats["active"])
	}
}

