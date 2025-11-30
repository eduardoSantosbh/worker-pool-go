package csvreader

import (
	"os"
	"testing"
	"time"
)

func createTempCSV(content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return "", err
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return "", err
	}

	return tmpfile.Name(), nil
}

func TestNewReader(t *testing.T) {
	reader := NewReader("test.csv")
	if reader == nil {
		t.Fatal("Expected reader instance, got nil")
	}
	if reader.filePath != "test.csv" {
		t.Errorf("Expected filePath 'test.csv', got '%s'", reader.filePath)
	}
}

func TestReadAll_ValidCSV(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,5500.00,TI,true,2024-01-15
Maria Santos,maria@empresa.com,32,6200.00,RH,true,2024-01-16`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) > 0 {
		t.Errorf("Expected no parse errors, got %d", len(parseErrors))
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	// Verifica primeiro registro
	rec1 := records[0]
	if rec1.Name != "João Silva" {
		t.Errorf("Expected name 'João Silva', got '%s'", rec1.Name)
	}
	if rec1.Email != "joao@empresa.com" {
		t.Errorf("Expected email 'joao@empresa.com', got '%s'", rec1.Email)
	}
	if rec1.Age != 28 {
		t.Errorf("Expected age 28, got %d", rec1.Age)
	}
	if rec1.Salary != 5500.00 {
		t.Errorf("Expected salary 5500.00, got %.2f", rec1.Salary)
	}
	if rec1.Department != "TI" {
		t.Errorf("Expected department 'TI', got '%s'", rec1.Department)
	}
	if !rec1.IsActive {
		t.Error("Expected IsActive true, got false")
	}
	if rec1.RowNumber != 2 {
		t.Errorf("Expected RowNumber 2, got %d", rec1.RowNumber)
	}
}

func TestReadAll_EmptyFile(t *testing.T) {
	filePath, err := createTempCSV("name,email,age,salary,department,is_active,created_at")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	// O código atual não retorna erro para CSV vazio, apenas 0 registros
	if err != nil {
		t.Logf("Note: ReadAll returned error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}

	if len(parseErrors) != 0 {
		t.Errorf("Expected 0 parse errors, got %d", len(parseErrors))
	}
}

func TestReadAll_FileNotFound(t *testing.T) {
	reader := NewReader("nonexistent.csv")
	records, parseErrors, err := reader.ReadAll()

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}

	if len(parseErrors) != 0 {
		t.Errorf("Expected 0 parse errors, got %d", len(parseErrors))
	}
}

func TestReadAll_InvalidColumns(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	// O CSV reader retorna erro quando há número incorreto de campos
	if err == nil {
		t.Error("Expected error for invalid number of columns, got nil")
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 valid records, got %d", len(records))
	}
	
	// parseErrors pode estar vazio se o erro ocorreu na leitura do CSV
	_ = parseErrors
}

func TestReadAll_InvalidAge(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,invalid,5500.00,TI,true,2024-01-15`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse error for invalid age, got none")
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 valid records, got %d", len(records))
	}
}

func TestReadAll_InvalidSalary(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,not_a_number,TI,true,2024-01-15`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	_, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse error for invalid salary, got none")
	}
}

func TestReadAll_InvalidDate(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,5500.00,TI,true,invalid-date`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	_, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse error for invalid date, got none")
	}
}

func TestReadAll_InvalidBoolean(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,5500.00,TI,maybe,2024-01-15`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	_, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse error for invalid boolean, got none")
	}
}

func TestReadAll_EmptyName(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
,joao@empresa.com,28,5500.00,TI,true,2024-01-15`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	_, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse error for empty name, got none")
	}
}

func TestReadAll_MultipleErrors(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,invalid,not_number,TI,maybe,invalid-date
,invalid-email,150,999999,InvalidDept,yes,bad-date`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parseErrors) < 2 {
		t.Errorf("Expected at least 2 parse errors, got %d", len(parseErrors))
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 valid records, got %d", len(records))
	}
}

func TestReadAll_MixedValidAndInvalid(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,5500.00,TI,true,2024-01-15
Invalid Name,,invalid,5500.00,TI,true,2024-01-16
Maria Santos,maria@empresa.com,32,6200.00,RH,true,2024-01-17`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, parseErrors, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 valid records, got %d", len(records))
	}

	if len(parseErrors) == 0 {
		t.Error("Expected parse errors, got none")
	}
}

func TestReadAll_DateParsing(t *testing.T) {
	csvContent := `name,email,age,salary,department,is_active,created_at
João Silva,joao@empresa.com,28,5500.00,TI,true,2024-01-15`

	filePath, err := createTempCSV(csvContent)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(filePath)

	reader := NewReader(filePath)
	records, _, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !records[0].CreatedAt.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, records[0].CreatedAt)
	}
}

