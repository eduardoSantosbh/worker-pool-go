package validator

import (
	"testing"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("Expected validator instance, got nil")
	}
}

func TestValidate_ValidRecord(t *testing.T) {
	v := NewValidator()
	record := &models.Record{
		Name:       "João Silva",
		Email:      "joao.silva@empresa.com",
		Age:        28,
		Salary:     5500.00,
		Department: "TI",
		IsActive:   true,
		RowNumber:  1,
	}

	err := v.Validate(record)
	if err != nil {
		t.Errorf("Expected no error for valid record, got %v", err)
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	v := NewValidator()
	testCases := []string{
		"email-invalido",
		"@empresa.com",
		"joao@",
		"joao@empresa",
		"",
		"joao.empresa.com",
	}

	for _, email := range testCases {
		t.Run(email, func(t *testing.T) {
			record := &models.Record{
				Name:       "João Silva",
				Email:      email,
				Age:        28,
				Salary:     5500.00,
				Department: "TI",
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if err == nil {
				t.Errorf("Expected error for invalid email %s, got nil", email)
			}
		})
	}
}

func TestValidate_Age(t *testing.T) {
	v := NewValidator()
	testCases := []struct {
		name    string
		age     int
		wantErr bool
	}{
		{"Age too young", 17, true},
		{"Age valid minimum", 18, false},
		{"Age valid", 30, false},
		{"Age valid maximum", 100, false},
		{"Age too old", 101, true},
		{"Age negative", -1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := &models.Record{
				Name:       "João Silva",
				Email:      "joao@empresa.com",
				Age:        tc.age,
				Salary:     5500.00,
				Department: "TI",
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for age %d, got nil", tc.age)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error for age %d, got %v", tc.age, err)
			}
		})
	}
}

func TestValidate_Salary(t *testing.T) {
	v := NewValidator()
	testCases := []struct {
		name    string
		salary  float64
		wantErr bool
	}{
		{"Salary too low", 999, true},
		{"Salary valid minimum", 1000, false},
		{"Salary valid", 5500, false},
		{"Salary valid maximum", 1000000, false},
		{"Salary too high", 1000001, true},
		{"Salary negative", -100, true},
		{"Salary zero", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := &models.Record{
				Name:       "João Silva",
				Email:      "joao@empresa.com",
				Age:        28,
				Salary:     tc.salary,
				Department: "TI",
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for salary %.2f, got nil", tc.salary)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error for salary %.2f, got %v", tc.salary, err)
			}
		})
	}
}

func TestValidate_Name(t *testing.T) {
	v := NewValidator()
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Name too short", "Jo", true},
		{"Name valid minimum", "Jos", false},
		{"Name valid", "João Silva Santos", false},
		{"Name too long", makeString(101), true},
		{"Name empty", "", true},
		{"Name only spaces", "   ", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := &models.Record{
				Name:       tc.value,
				Email:      "joao@empresa.com",
				Age:        28,
				Salary:     5500.00,
				Department: "TI",
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if tc.wantErr && err == nil {
				t.Errorf("Expected error for name '%s', got nil", tc.value)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error for name '%s', got %v", tc.value, err)
			}
		})
	}
}

func TestValidate_Department(t *testing.T) {
	v := NewValidator()
	validDepartments := []string{"TI", "RH", "Financeiro", "Vendas", "Marketing", "Operações", "Jurídico", "Administração"}
	invalidDepartments := []string{"Vendas", "Recursos Humanos", "IT", "tech", ""}

	// Testa departamentos válidos
	for _, dept := range validDepartments {
		t.Run("Valid_"+dept, func(t *testing.T) {
			record := &models.Record{
				Name:       "João Silva",
				Email:      "joao@empresa.com",
				Age:        28,
				Salary:     5500.00,
				Department: dept,
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if err != nil {
				t.Errorf("Expected no error for valid department %s, got %v", dept, err)
			}
		})
	}

	// Testa departamentos inválidos
	for _, dept := range invalidDepartments {
		// Pula se estiver na lista de válidos
		isValid := false
		for _, valid := range validDepartments {
			if dept == valid {
				isValid = true
				break
			}
		}
		if isValid {
			continue
		}

		t.Run("Invalid_"+dept, func(t *testing.T) {
			record := &models.Record{
				Name:       "João Silva",
				Email:      "joao@empresa.com",
				Age:        28,
				Salary:     5500.00,
				Department: dept,
				IsActive:   true,
				RowNumber:  1,
			}

			err := v.Validate(record)
			if err == nil {
				t.Errorf("Expected error for invalid department '%s', got nil", dept)
			}
		})
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	v := NewValidator()
	record := &models.Record{
		Name:       "Jo", // Muito curto
		Email:      "email-invalido", // Email inválido
		Age:        150, // Muito velho
		Salary:     500, // Muito baixo
		Department: "DepartamentoInvalido", // Inválido
		IsActive:   true,
		RowNumber:  1,
	}

	err := v.Validate(record)
	if err == nil {
		t.Fatal("Expected error for record with multiple validation issues, got nil")
	}

	// Verifica se a mensagem de erro contém informações sobre os problemas
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}
}

func makeString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

