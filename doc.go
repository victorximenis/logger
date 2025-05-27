// Package logger fornece uma interface unificada para logging estruturado
// que pode ser implementada por diferentes bibliotecas de logging.
//
// O pacote utiliza o padrão Adapter para permitir que diferentes
// implementações de logging (como logrus, zap, slog, etc.) sejam
// utilizadas através de uma interface comum.
//
// Características principais:
//
//   - Interface fluente com method chaining
//   - Suporte a logging estruturado com campos tipados
//   - Níveis de log padrão (DEBUG, INFO, WARN, ERROR, FATAL)
//   - Propagação de contexto
//   - Campos pré-definidos por instância
//
// Exemplo de uso básico:
//
//	// Criar um adapter (implementação específica)
//	adapter := &MyLoggerAdapter{}
//
//	// Criar o logger
//	log := logger.New(adapter)
//
//	// Usar o logger com method chaining
//	log.Info(ctx).
//		Str("user_id", "123").
//		Int("attempt", 1).
//		Msg("User login successful")
//
// Exemplo com campos pré-definidos:
//
//	// Logger com campos comuns
//	userLogger := log.WithFields(map[string]interface{}{
//		"service": "auth",
//		"version": "1.0.0",
//	})
//
//	// Todos os logs incluirão os campos pré-definidos
//	userLogger.Error(ctx).
//		Err(err).
//		Msg("Authentication failed")
//
// Para implementar um novo adapter, implemente a interface core.LoggerAdapter:
//
//	type MyAdapter struct {
//		// campos necessários
//	}
//
//	func (a *MyAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
//		// implementação específica
//	}
//
//	func (a *MyAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
//		// retornar nova instância com contexto
//	}
//
//	func (a *MyAdapter) IsLevelEnabled(level core.Level) bool {
//		// verificar se o nível está habilitado
//	}
package logger
