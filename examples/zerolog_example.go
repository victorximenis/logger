package main

import (
	"context"
	"errors"
	"os"

	"github.com/victorximenis/logger"
	"github.com/victorximenis/logger/adapters"
	"github.com/victorximenis/logger/core"
)

func zerologExample() {
	// Exemplo 1: Configuração básica para produção
	productionConfig := &adapters.ZerologConfig{
		Writer:        os.Stdout,
		Level:         core.INFO,
		TimeFormat:    "", // Usa timestamp Unix por padrão
		PrettyPrint:   false,
		CallerEnabled: false,
	}

	prodAdapter := adapters.NewZerologAdapter(productionConfig)
	prodLogger := logger.New(prodAdapter)

	ctx := context.Background()

	// Log básico
	prodLogger.Info(ctx).
		Str("service", "user-service").
		Str("version", "1.0.0").
		Msg("Service started successfully")

	// Log com erro
	err := errors.New("database connection failed")
	prodLogger.Error(ctx).
		Err(err).
		Str("database", "postgres").
		Int("retry_count", 3).
		Msg("Failed to connect to database")

	// Exemplo 2: Configuração para desenvolvimento
	devConfig := &adapters.ZerologConfig{
		Writer:        os.Stdout,
		Level:         core.DEBUG,
		PrettyPrint:   true, // Formatação legível
		CallerEnabled: true, // Mostra arquivo e linha
	}

	devAdapter := adapters.NewZerologAdapter(devConfig)
	devLogger := logger.New(devAdapter)

	// Log de debug em desenvolvimento
	devLogger.Debug(ctx).
		Str("function", "processUser").
		Int("user_id", 123).
		Float64("processing_time", 0.045).
		Msg("Processing user data")

	// Exemplo 3: Logger com campos pré-definidos
	serviceLogger := prodLogger.WithFields(map[string]interface{}{
		"service":    "auth-service",
		"version":    "2.1.0",
		"deployment": "production",
	})

	// Todos os logs deste logger incluirão os campos pré-definidos
	serviceLogger.Info(ctx).
		Str("user_id", "user-123").
		Str("action", "login").
		Bool("success", true).
		Msg("User authentication successful")

	serviceLogger.Warn(ctx).
		Str("user_id", "user-456").
		Str("action", "login").
		Bool("success", false).
		Str("reason", "invalid_password").
		Int("attempt_count", 3).
		Msg("User authentication failed")

	// Exemplo 4: Usando diferentes níveis de log
	serviceLogger.Debug(ctx).Msg("Debug information for troubleshooting")
	serviceLogger.Info(ctx).Msg("General information about application flow")
	serviceLogger.Warn(ctx).Msg("Warning about potential issues")
	serviceLogger.Error(ctx).Msg("Error that doesn't stop execution")

	// Exemplo 5: Log com contexto
	userCtx := context.WithValue(ctx, "user_id", "user-789")
	userCtx = context.WithValue(userCtx, "session_id", "session-abc123")

	contextLogger := serviceLogger.WithContext(userCtx)
	contextLogger.Info(ctx).
		Str("operation", "update_profile").
		Any("changes", map[string]string{
			"email": "new@example.com",
			"name":  "New Name",
		}).
		Msg("User profile updated")

	// Exemplo 6: Usando Msgf para formatação
	serviceLogger.Info(ctx).
		Str("user_id", "user-999").
		Msgf("User %s performed %d actions in the last %d minutes", "john_doe", 15, 30)

	// Exemplo 7: Log apenas com campos (sem mensagem)
	serviceLogger.Info(ctx).
		Str("event_type", "metric").
		Str("metric_name", "response_time").
		Float64("value", 0.125).
		Str("unit", "seconds").
		Send()
}
