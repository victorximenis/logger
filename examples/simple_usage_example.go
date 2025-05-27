//go:build ignore

package main

import (
	"context"
	"log"

	"github.com/victorximenis/logger"
)

func simpleUsageExample() {
	// Inicialização simples com configuração padrão
	err := logger.InitFromEnv()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	ctx := context.Background()

	// Uso básico do logger
	logger.Info(ctx).
		Str("service", "example").
		Str("version", "1.0.0").
		Msg("Application started successfully")

	// Exemplo com campos dinâmicos
	userID := "user123"
	logger.Info(ctx).
		Str("user_id", userID).
		Int("login_attempts", 3).
		Bool("success", true).
		Msg("User authentication completed")

	// Exemplo de erro
	logger.Error(ctx).
		Str("operation", "database_connection").
		Str("error", "connection timeout").
		Msg("Failed to connect to database")

	// Exemplo com contexto personalizado
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"component": "auth",
		"module":    "login",
	})

	loggerWithFields.Debug(ctx).
		Str("step", "validate_credentials").
		Msg("Validating user credentials")

	logger.Info(ctx).Msg("Application finished")
}
