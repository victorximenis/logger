package main

import (
	"context"
	"fmt"
	"os"

	"github.com/victorximenis/logger"
	"github.com/victorximenis/logger/core"
)

func configExample() {
	fmt.Println("=== Exemplo de Configuração do Logger ===")

	// Exemplo 1: Configuração manual
	fmt.Println("1. Configuração Manual:")
	config := logger.Config{
		ServiceName:   "exemplo-service",
		Environment:   "development",
		Output:        logger.OutputStdout,
		LogLevel:      core.INFO,
		LogFilePath:   "/tmp/exemplo.log",
		TenantID:      "tenant-123",
		PrettyPrint:   true,
		CallerEnabled: true,
	}

	// Inicializar logger com configuração manual
	if err := logger.Init(config); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	ctx := context.Background()
	logger.Info(ctx).Str("exemplo", "configuracao_manual").Msg("Logger inicializado com configuração manual")

	fmt.Printf("Configuração atual: %s\n\n", logger.GetConfig().String())

	// Exemplo 2: Configuração via variáveis de ambiente
	fmt.Println("2. Configuração via Variáveis de Ambiente:")

	// Definir variáveis de ambiente
	os.Setenv("LOGGER_SERVICE_NAME", "env-service")
	os.Setenv("LOGGER_ENVIRONMENT", "production")
	os.Setenv("LOGGER_OUTPUT", "stdout")
	os.Setenv("LOGGER_LOG_LEVEL", "debug")
	os.Setenv("LOGGER_TENANT_ID", "tenant-456")
	os.Setenv("LOGGER_PRETTY_PRINT", "false")
	os.Setenv("LOGGER_CALLER_ENABLED", "true")

	// Carregar configuração de variáveis de ambiente
	envConfig := logger.LoadConfigFromEnv()
	fmt.Printf("Configuração carregada do ambiente: %s\n", envConfig.String())

	// Inicializar com configuração do ambiente
	if err := logger.Init(envConfig); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	logger.Debug(ctx).Str("exemplo", "configuracao_ambiente").Msg("Logger inicializado com configuração do ambiente")
	logger.Warn(ctx).Str("tenant", envConfig.TenantID).Msg("Exemplo de log com tenant ID")

	// Exemplo 3: Usando helpers globais
	fmt.Println("\n3. Usando Helpers Globais:")

	// Usar funções helper globais
	logger.Info(ctx).Str("tipo", "helper_global").Msg("Usando função Info global")
	logger.Error(ctx).Str("codigo", "ERR001").Msg("Exemplo de erro usando helper global")

	// Usar logger com contexto
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info(ctx).Str("contexto", "presente").Msg("Logger com contexto")

	// Usar logger com campos pré-definidos
	fieldsLogger := logger.WithFields(map[string]interface{}{
		"modulo":  "exemplo",
		"versao":  "1.0.0",
		"usuario": "admin",
	})
	fieldsLogger.Info(ctx).Msg("Logger com campos pré-definidos")

	// Exemplo 4: Configuração com múltiplas saídas
	fmt.Println("\n4. Configuração com Múltiplas Saídas:")

	multiConfig := logger.Config{
		ServiceName:   "multi-output-service",
		Environment:   "staging",
		Output:        logger.OutputStdout | logger.OutputFile, // stdout E file
		LogLevel:      core.WARN,
		LogFilePath:   "/tmp/multi-output.log",
		TenantID:      "",
		PrettyPrint:   false,
		CallerEnabled: false,
	}

	if err := logger.Init(multiConfig); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	logger.Warn(ctx).Str("saida", "multipla").Msg("Este log vai para stdout E arquivo")
	logger.Error(ctx).Str("tipo", "critico").Msg("Log crítico em múltiplas saídas")

	// Exemplo 5: Validação de configuração
	fmt.Println("\n5. Validação de Configuração:")

	// Configuração inválida
	invalidConfig := logger.Config{
		ServiceName: "", // inválido - não pode ser vazio
		Environment: "test",
		Output:      logger.OutputFile,
		LogLevel:    core.INFO,
		LogFilePath: "", // inválido - obrigatório quando Output inclui file
	}

	if err := invalidConfig.Validate(); err != nil {
		fmt.Printf("Configuração inválida detectada: %v\n", err)
	}

	// Configuração válida
	validConfig := logger.NewConfig() // usa valores padrão
	if err := validConfig.Validate(); err != nil {
		fmt.Printf("Erro inesperado: %v\n", err)
	} else {
		fmt.Println("Configuração padrão é válida")
	}

	// Exemplo 6: Inicialização automática
	fmt.Println("\n6. Inicialização Automática:")

	// Reset do estado global para demonstrar inicialização automática
	// (Normalmente não seria necessário em código real)

	// GetLogger() inicializa automaticamente se não foi inicializado
	autoLogger := logger.GetLogger()
	autoLogger.Info(ctx).Str("tipo", "automatico").Msg("Logger inicializado automaticamente")

	fmt.Printf("Logger foi inicializado automaticamente: %t\n", logger.IsInitialized())

	fmt.Println("\n=== Exemplo Concluído ===")

	// Limpar variáveis de ambiente
	os.Unsetenv("LOGGER_SERVICE_NAME")
	os.Unsetenv("LOGGER_ENVIRONMENT")
	os.Unsetenv("LOGGER_OUTPUT")
	os.Unsetenv("LOGGER_LOG_LEVEL")
	os.Unsetenv("LOGGER_TENANT_ID")
	os.Unsetenv("LOGGER_PRETTY_PRINT")
	os.Unsetenv("LOGGER_CALLER_ENABLED")
}
