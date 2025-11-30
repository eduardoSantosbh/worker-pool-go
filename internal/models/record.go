package models

import (
	"fmt"
	"time"
)

// Record representa um registro do CSV após validação
type Record struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Age         int       `json:"age"`
	Salary      float64   `json:"salary"`
	Department  string    `json:"department"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at"`
	RowNumber   int       `json:"row_number"` // Linha original do CSV
}

// ValidationError representa um erro de validação
type ValidationError struct {
	RowNumber int
	Field     string
	Message   string
	Value     interface{}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Linha %d, Campo '%s': %s (Valor: %v)", e.RowNumber, e.Field, e.Message, e.Value)
}

// GetName retorna o nome do registro (para logs)
func (r *Record) GetName() string {
	return r.Name
}

// ProcessingResult representa o resultado do processamento de um registro
type ProcessingResult struct {
	RowNumber int
	Record    *Record
	Success   bool
	Error     error
	Duration  time.Duration
}
