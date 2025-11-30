package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
)

// Validator valida registros
type Validator struct {
	emailRegex *regexp.Regexp
}

// NewValidator cria uma nova instância do validador
func NewValidator() *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return &Validator{
		emailRegex: emailRegex,
	}
}

// Validate valida um registro
func (v *Validator) Validate(record *models.Record) error {
	var errors []string

	// Validação de email
	if !v.emailRegex.MatchString(record.Email) {
		errors = append(errors, fmt.Sprintf("email inválido: %s", record.Email))
	}

	// Validação de idade
	if record.Age < 18 || record.Age > 100 {
		errors = append(errors, fmt.Sprintf("idade fora do range válido (18-100): %d", record.Age))
	}

	// Validação de salário
	if record.Salary < 1000 || record.Salary > 1000000 {
		errors = append(errors, fmt.Sprintf("salário fora do range válido (1000-1000000): %.2f", record.Salary))
	}

	// Validação de nome
	name := strings.TrimSpace(record.Name)
	if len(name) < 3 || len(name) > 100 {
		errors = append(errors, fmt.Sprintf("nome deve ter entre 3 e 100 caracteres: %s", name))
	}

	// Validação de departamento
	departments := map[string]bool{
		"TI":            true,
		"RH":            true,
		"Financeiro":    true,
		"Vendas":        true,
		"Marketing":     true,
		"Operações":     true,
		"Jurídico":      true,
		"Administração": true,
	}

	department := strings.TrimSpace(record.Department)
	if !departments[department] {
		errors = append(errors, fmt.Sprintf("departamento inválido: %s", department))
	}

	if len(errors) > 0 {
		return &models.ValidationError{
			RowNumber: record.RowNumber,
			Field:     "validação",
			Message:   strings.Join(errors, "; "),
			Value:     record,
		}
	}

	return nil
}
