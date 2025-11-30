package csvreader

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
)

// Reader lê e processa arquivos CSV
type Reader struct {
	filePath string
}

// NewReader cria uma nova instância do leitor CSV
func NewReader(filePath string) *Reader {
	return &Reader{
		filePath: filePath,
	}
}

// ReadAll lê todo o arquivo CSV e retorna os registros
func (r *Reader) ReadAll() ([]*models.Record, []error, error) {
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao abrir arquivo: %w", err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	csvReader.Comma = ','
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true

	// Lê todas as linhas
	rows, err := csvReader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao ler CSV: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("arquivo CSV vazio")
	}

	// Pula o cabeçalho (primeira linha)
	rows = rows[1:]

	var records []*models.Record
	var errors []error

	// Processa cada linha
	for i, row := range rows {
		rowNumber := i + 2 // +2 porque pulamos header e índice começa em 0
		record, err := r.parseRow(row, rowNumber)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		records = append(records, record)
	}

	return records, errors, nil
}

// parseRow converte uma linha do CSV em um Record
func (r *Reader) parseRow(row []string, rowNumber int) (*models.Record, error) {
	if len(row) < 7 {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "estrutura",
			Message:   "número insuficiente de colunas",
			Value:     len(row),
		}
	}

	// Nome
	name := row[0]
	if name == "" {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "name",
			Message:   "nome não pode ser vazio",
			Value:     name,
		}
	}

	// Email
	email := row[1]
	if email == "" {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "email",
			Message:   "email não pode ser vazio",
			Value:     email,
		}
	}

	// Age
	age, err := strconv.Atoi(row[2])
	if err != nil || age < 0 || age > 150 {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "age",
			Message:   "idade inválida (deve ser entre 0 e 150)",
			Value:     row[2],
		}
	}

	// Salary
	salary, err := strconv.ParseFloat(row[3], 64)
	if err != nil || salary < 0 {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "salary",
			Message:   "salário inválido (deve ser um número positivo)",
			Value:     row[3],
		}
	}

	// Department
	department := row[4]
	if department == "" {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "department",
			Message:   "departamento não pode ser vazio",
			Value:     department,
		}
	}

	// IsActive
	isActive, err := strconv.ParseBool(row[5])
	if err != nil {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "is_active",
			Message:   "valor inválido (deve ser true ou false)",
			Value:     row[5],
		}
	}

	// CreatedAt
	createdAt, err := time.Parse("2006-01-02", row[6])
	if err != nil {
		return nil, &models.ValidationError{
			RowNumber: rowNumber,
			Field:     "created_at",
			Message:   "data inválida (formato esperado: YYYY-MM-DD)",
			Value:     row[6],
		}
	}

	return &models.Record{
		Name:        name,
		Email:       email,
		Age:         age,
		Salary:      salary,
		Department:  department,
		IsActive:    isActive,
		CreatedAt:   createdAt,
		ProcessedAt: time.Now(),
		RowNumber:   rowNumber,
	}, nil
}
