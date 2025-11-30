package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/seu-usuario/worker-pool-csv-processor/internal/models"
)

// DB gerencia a conexão com o banco de dados
type DB struct {
	conn *sql.DB
}

// NewDB cria uma nova instância do banco de dados
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco de dados: %w", err)
	}

	db := &DB{conn: conn}

	// Cria as tabelas se não existirem
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("erro ao criar tabelas: %w", err)
	}

	return db, nil
}

// createTables cria as tabelas necessárias
func (d *DB) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS employees (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		age INTEGER NOT NULL,
		salary REAL NOT NULL,
		department TEXT NOT NULL,
		is_active BOOLEAN NOT NULL,
		created_at TIMESTAMP NOT NULL,
		processed_at TIMESTAMP NOT NULL,
		row_number INTEGER,
		created_at_db TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_email ON employees(email);
	CREATE INDEX IF NOT EXISTS idx_department ON employees(department);
	CREATE INDEX IF NOT EXISTS idx_is_active ON employees(is_active);
	`

	_, err := d.conn.Exec(query)
	return err
}

// InsertRecord insere um registro no banco de dados
func (d *DB) InsertRecord(record *models.Record) error {
	query := `
	INSERT INTO employees (name, email, age, salary, department, is_active, created_at, processed_at, row_number)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(email) DO UPDATE SET
		name = excluded.name,
		age = excluded.age,
		salary = excluded.salary,
		department = excluded.department,
		is_active = excluded.is_active,
		processed_at = excluded.processed_at
	`

	_, err := d.conn.Exec(
		query,
		record.Name,
		record.Email,
		record.Age,
		record.Salary,
		record.Department,
		record.IsActive,
		record.CreatedAt,
		record.ProcessedAt,
		record.RowNumber,
	)

	if err != nil {
		return fmt.Errorf("erro ao inserir registro: %w", err)
	}

	return nil
}

// GetStats retorna estatísticas do banco de dados
func (d *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total de registros
	var total int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM employees").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	// Total por departamento
	rows, err := d.conn.Query(`
		SELECT department, COUNT(*) as count 
		FROM employees 
		GROUP BY department
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDepartment := make(map[string]int)
	for rows.Next() {
		var dept string
		var count int
		if err := rows.Scan(&dept, &count); err != nil {
			return nil, err
		}
		byDepartment[dept] = count
	}
	stats["by_department"] = byDepartment

	// Total ativos/inativos
	var active int
	err = d.conn.QueryRow("SELECT COUNT(*) FROM employees WHERE is_active = 1").Scan(&active)
	if err != nil {
		return nil, err
	}
	stats["active"] = active
	stats["inactive"] = total - active

	return stats, nil
}

// Close fecha a conexão com o banco de dados
func (d *DB) Close() error {
	return d.conn.Close()
}

// Cleanup remove todos os registros (útil para testes)
func (d *DB) Cleanup() error {
	_, err := d.conn.Exec("DELETE FROM employees")
	return err
}

// GetRecordByEmail busca um registro por email
func (d *DB) GetRecordByEmail(email string) (*models.Record, error) {
	query := `
		SELECT id, name, email, age, salary, department, is_active, created_at, processed_at, row_number
		FROM employees
		WHERE email = ?
	`

	var record models.Record
	var createdAtStr, processedAtStr string

	err := d.conn.QueryRow(query, email).Scan(
		&record.ID,
		&record.Name,
		&record.Email,
		&record.Age,
		&record.Salary,
		&record.Department,
		&record.IsActive,
		&createdAtStr,
		&processedAtStr,
		&record.RowNumber,
	)

	if err != nil {
		return nil, err
	}

	record.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	record.ProcessedAt, _ = time.Parse("2006-01-02 15:04:05", processedAtStr)

	return &record, nil
}
