// Package logger fornece uma interface unificada para logging estruturado
// que pode ser implementada por diferentes bibliotecas de logging.
//
// O pacote utiliza o padrão Adapter para permitir que diferentes
// implementações de logging (como logrus, zap, slog, etc.) sejam
// utilizadas através de uma interface comum, com suporte completo
// a observabilidade, middlewares HTTP e conformidade LGPD.
//
// Características principais:
//
//   - Interface fluente com method chaining
//   - Suporte a logging estruturado com campos tipados
//   - Níveis de log padrão (DEBUG, INFO, WARN, ERROR, FATAL)
//   - Propagação de contexto
//   - Campos pré-definidos por instância
//   - Logger global thread-safe
//   - Perfis de configuração pré-definidos (development, staging, production)
//   - Observabilidade integrada (Datadog, ELK Stack)
//   - Middlewares para frameworks HTTP (Gin, Fiber, Chi)
//   - Integração com PostgreSQL via PGX
//   - Sanitização LGPD automática
//   - Correlation IDs e distributed tracing
//
// # Uso Básico
//
// Inicialização rápida com perfis pré-configurados:
//
//	// Inicialização para produção (Datadog + ELK habilitados)
//	err := logger.InitWithProfile("production", "my-service")
//	if err != nil {
//		panic(err)
//	}
//
//	ctx := context.Background()
//
//	// Usar o logger global
//	logger.Info(ctx).
//		Str("user_id", "123").
//		Int("attempt", 1).
//		Msg("User login successful")
//
// # Configuração Manual
//
// Para configuração personalizada:
//
//	config := logger.NewConfig()
//	config.ServiceName = "auth-service"
//	config.Environment = "production"
//	config.LogLevel = logger.INFO
//
//	// Habilitar observabilidade
//	config.Observability.Enabled = true
//	config.Observability.EnableDatadog = true
//	config.Observability.EnableELK = true
//
//	err := logger.Init(config)
//	if err != nil {
//		panic(err)
//	}
//
// # Logger com Campos Pré-definidos
//
// Criar instâncias com campos comuns:
//
//	// Logger com campos comuns
//	userLogger := logger.WithFields(map[string]interface{}{
//		"service": "auth",
//		"version": "1.0.0",
//	})
//
//	// Todos os logs incluirão os campos pré-definidos
//	userLogger.Error(ctx).
//		Err(err).
//		Msg("Authentication failed")
//
// # Middlewares HTTP
//
// O pacote inclui middlewares para frameworks populares:
//
//	// Gin
//	r.Use(middlewares.GinLogger())
//
//	// Fiber
//	app.Use(middlewares.FiberLogger())
//
//	// Chi
//	r.Use(middlewares.ChiLogger())
//
// # Implementando um Adapter Personalizado
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
//
// # Configuração via Variáveis de Ambiente
//
// O pacote suporta configuração através de variáveis de ambiente:
//
//   - LOGGER_SERVICE_NAME: Nome do serviço
//   - LOGGER_ENVIRONMENT: Ambiente (development, staging, production)
//   - LOGGER_LOG_LEVEL: Nível de log (debug, info, warn, error, fatal)
//   - LOGGER_OUTPUT: Tipo de saída (stdout, file)
//   - LOGGER_PRETTY_PRINT: Formatação legível (true/false)
//   - LOGGER_OBSERVABILITY_ENABLED: Habilitar observabilidade (true/false)
//
// Para carregar configuração do ambiente:
//
//	err := logger.InitFromEnv()
//	if err != nil {
//		panic(err)
//	}
//
// # Observabilidade
//
// O pacote inclui integração nativa com sistemas de observabilidade:
//
//   - Datadog APM para distributed tracing
//   - ELK Stack para agregação de logs
//   - Correlation IDs automáticos
//   - Métricas de performance
//   - Dashboards pré-configurados
//
// # Conformidade LGPD
//
// Sanitização automática de dados sensíveis:
//
//   - CPF, CNPJ, emails, telefones
//   - Dados de cartão de crédito
//   - Senhas e tokens
//   - Configuração flexível de campos sensíveis
package logger
