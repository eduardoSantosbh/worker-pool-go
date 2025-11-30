.PHONY: help build run test clean stats

help: ## Mostra esta mensagem de ajuda
	@echo "Comandos disponíveis:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Compila o projeto
	@echo "🔨 Compilando..."
	go build -o bin/processor ./cmd/processor
	@echo "✅ Compilado em bin/processor"

run: build ## Compila e executa o processador
	@echo "🚀 Executando processador..."
	./bin/processor -csv data/employees.csv -db employees.db

run-large: build ## Executa com mais workers
	./bin/processor -csv data/employees.csv -db employees.db -workers 8 -queue 200

stats: build ## Mostra estatísticas do banco
	./bin/processor -db employees.db -stats

test: ## Executa testes
	@echo "🧪 Executando testes..."
	go test -v ./...

test-cover: ## Executa testes com coverage
	@echo "🧪 Executando testes com coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Relatório gerado: coverage.html"

clean: ## Limpa arquivos gerados
	@echo "🧹 Limpando..."
	rm -rf bin/
	rm -f *.db
	rm -f coverage.out coverage.html
	@echo "✅ Limpeza concluída!"

fmt: ## Formata o código
	@echo "📝 Formatando código..."
	go fmt ./...

lint: ## Executa linter (requer golangci-lint)
	@echo "🔍 Executando linter..."
	golangci-lint run

deps: ## Instala dependências
	@echo "📦 Instalando dependências..."
	go mod download
	go mod tidy

setup: ## Setup inicial do projeto
	@echo "⚙️  Configurando projeto..."
	go mod download
	go mod tidy
	mkdir -p bin data
	@echo "✅ Setup concluído!"

all: deps test build ## Executa tudo: dependências, testes e build

