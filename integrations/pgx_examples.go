package integrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/victorximenis/logger/core"
)

// ExampleBasicUsage demonstra o uso básico da integração PGX
func ExampleBasicUsage() {
	// Criar um logger adapter (exemplo usando um mock)
	logger := &mockLoggerAdapter{}

	// Exemplo 1: Pool com configurações padrão
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	pool, err := NewPgxPoolWithDefaults(ctx, connString, logger)
	if err != nil {
		fmt.Printf("Erro ao criar pool: %v\n", err)
		return
	}
	defer pool.Close()

	// O pool agora está configurado com logging automático
	// Todas as operações de banco serão logadas
}

// ExampleProductionUsage demonstra configuração para produção
func ExampleProductionUsage() {
	logger := &mockLoggerAdapter{}
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	// Pool otimizado para produção
	pool, err := NewPgxPoolProduction(ctx, connString, logger)
	if err != nil {
		fmt.Printf("Erro ao criar pool de produção: %v\n", err)
		return
	}
	defer pool.Close()

	// Configurações de produção:
	// - Apenas logs de WARNING e ERROR
	// - Queries e argumentos sanitizados
	// - Queries truncadas em 500 caracteres
}

// ExampleDevelopmentUsage demonstra configuração para desenvolvimento
func ExampleDevelopmentUsage() {
	logger := &mockLoggerAdapter{}
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	// Pool otimizado para desenvolvimento
	pool, err := NewPgxPoolDevelopment(ctx, connString, logger)
	if err != nil {
		fmt.Printf("Erro ao criar pool de desenvolvimento: %v\n", err)
		return
	}
	defer pool.Close()

	// Configurações de desenvolvimento:
	// - Logs detalhados (DEBUG level)
	// - Queries e argumentos não sanitizados
	// - Queries maiores (2000 caracteres)
}

// ExampleCustomConfiguration demonstra configuração personalizada
func ExampleCustomConfiguration() {
	logger := &mockLoggerAdapter{}
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	// Configuração personalizada usando fluent API
	config := DefaultPgxLoggerConfig(logger).
		WithMinLevel(tracelog.LogLevelInfo).
		WithSanitizeQueries(true).
		WithSanitizeArgs(false). // Mostrar argumentos mas sanitizar queries
		WithMaxQueryLength(1500)

	pool, err := NewPgxPool(ctx, connString, config)
	if err != nil {
		fmt.Printf("Erro ao criar pool personalizado: %v\n", err)
		return
	}
	defer pool.Close()
}

// ExampleWithOptions demonstra uso com opções estruturadas
func ExampleWithOptions() {
	logger := &mockLoggerAdapter{}
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	// Configuração usando struct de opções
	options := DefaultPgxPoolOptions()
	options.LogLevel = tracelog.LogLevelDebug
	options.Production = false
	options.MaxQueryLength = 2000

	// Configuração personalizada adicional
	options.CustomConfig = func(config *PgxLoggerConfig) {
		// Personalizar ainda mais se necessário
		config.SanitizeQueries = false
	}

	pool, err := NewPgxPoolWithOptions(ctx, connString, logger, options)
	if err != nil {
		fmt.Printf("Erro ao criar pool com opções: %v\n", err)
		return
	}
	defer pool.Close()
}

// ExampleManualConfiguration demonstra configuração manual de um pool existente
func ExampleManualConfiguration() {
	logger := &mockLoggerAdapter{}
	ctx := context.Background()
	connString := "postgres://user:password@localhost:5432/database"

	// Criar configuração do pool manualmente
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		fmt.Printf("Erro ao parsear config: %v\n", err)
		return
	}

	// Configurar outras opções do pool se necessário
	config.MaxConns = 10
	config.MinConns = 2

	// Adicionar logging ao pool existente
	loggerConfig := DefaultPgxLoggerConfig(logger).
		WithMinLevel(tracelog.LogLevelInfo).
		WithSanitizeQueries(true)

	ConfigurePgxPool(config, loggerConfig)

	// Criar pool com configuração personalizada
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		fmt.Printf("Erro ao criar pool: %v\n", err)
		return
	}
	defer pool.Close()
}

// mockLoggerAdapter é um mock simples para demonstração
type mockLoggerAdapter struct{}

func (m *mockLoggerAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	fmt.Printf("[%s] %s - %v\n", level.String(), msg, fields)
}

func (m *mockLoggerAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	return m
}

func (m *mockLoggerAdapter) IsLevelEnabled(level core.Level) bool {
	return true
}
