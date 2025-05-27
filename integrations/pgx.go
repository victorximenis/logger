package integrations

import (
	"context"
	"regexp"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/victorximenis/logger/core"
	"github.com/victorximenis/logger/sanitize"
)

var (
	// Cache de padrões regex para sanitização de queries
	queryRegexCache = make(map[string]*regexp.Regexp)
	queryRegexMutex sync.RWMutex
)

// PgxLoggerConfig define a configuração para o logger PGX
type PgxLoggerConfig struct {
	// MinLevel define o nível mínimo de log
	MinLevel tracelog.LogLevel
	// SanitizeQueries habilita sanitização de queries SQL
	SanitizeQueries bool
	// SanitizeArgs habilita sanitização de argumentos de query
	SanitizeArgs bool
	// MaxQueryLength define o tamanho máximo da query para logging
	MaxQueryLength int
	// Logger define o logger adapter a ser usado
	Logger core.LoggerAdapter
}

// DefaultPgxLoggerConfig retorna uma configuração padrão para o logger PGX
func DefaultPgxLoggerConfig(logger core.LoggerAdapter) PgxLoggerConfig {
	return PgxLoggerConfig{
		MinLevel:        tracelog.LogLevelInfo,
		SanitizeQueries: true,
		SanitizeArgs:    true,
		MaxQueryLength:  1000,
		Logger:          logger,
	}
}

// PgxLogger implementa a interface tracelog.Logger do PGX
type PgxLogger struct {
	config PgxLoggerConfig
}

// NewPgxLogger cria uma nova instância do logger PGX
func NewPgxLogger(config PgxLoggerConfig) *PgxLogger {
	return &PgxLogger{
		config: config,
	}
}

// Log implementa a interface tracelog.Logger
func (pl *PgxLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	// Verificar se o nível está habilitado
	if level < pl.config.MinLevel {
		return
	}

	// Mapear nível do PGX para nível do nosso logger
	logLevel := pl.mapLogLevel(level)

	// Preparar campos do log
	fields := make(map[string]interface{})
	fields["component"] = "pgx"
	fields["level"] = level.String()

	// Processar dados do PGX
	for k, v := range data {
		switch k {
		case "sql":
			// Sanitizar query SQL se habilitado
			if pl.config.SanitizeQueries {
				if sqlStr, ok := v.(string); ok {
					fields[k] = pl.sanitizeQuery(sqlStr)
				} else {
					fields[k] = v
				}
			} else {
				// Truncar query se muito longa
				if sqlStr, ok := v.(string); ok && len(sqlStr) > pl.config.MaxQueryLength {
					fields[k] = sqlStr[:pl.config.MaxQueryLength] + "..."
				} else {
					fields[k] = v
				}
			}
		case "args":
			// Sanitizar argumentos se habilitado
			if pl.config.SanitizeArgs {
				fields[k] = "[REDACTED]"
			} else {
				fields[k] = v
			}
		case "time":
			// Converter duração para milliseconds se for time.Duration
			if duration, ok := v.(interface{ Milliseconds() int64 }); ok {
				fields["duration_ms"] = duration.Milliseconds()
			} else {
				fields[k] = v
			}
		default:
			fields[k] = v
		}
	}

	// Fazer o log
	pl.config.Logger.Log(ctx, logLevel, msg, fields)
}

// mapLogLevel mapeia níveis do PGX para níveis do nosso logger
func (pl *PgxLogger) mapLogLevel(level tracelog.LogLevel) core.Level {
	switch level {
	case tracelog.LogLevelTrace:
		return core.DEBUG
	case tracelog.LogLevelDebug:
		return core.DEBUG
	case tracelog.LogLevelInfo:
		return core.INFO
	case tracelog.LogLevelWarn:
		return core.WARN
	case tracelog.LogLevelError:
		return core.ERROR
	default:
		return core.INFO
	}
}

// sanitizeQuery sanitiza uma query SQL removendo dados sensíveis
func (pl *PgxLogger) sanitizeQuery(query string) string {
	if !pl.config.SanitizeQueries {
		return query
	}

	// Truncar se muito longa
	if len(query) > pl.config.MaxQueryLength {
		query = query[:pl.config.MaxQueryLength] + "..."
	}

	// Usar sistema de sanitização existente
	config := sanitize.DefaultSensitiveFieldConfig()
	sanitized := sanitize.SanitizeString(query, config)

	// Aplicar padrões específicos para SQL
	sanitized = pl.applySQLSanitization(sanitized)

	return sanitized
}

// applySQLSanitization aplica sanitização específica para SQL
func (pl *PgxLogger) applySQLSanitization(query string) string {
	// Padrões para sanitizar valores em queries SQL
	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Strings entre aspas simples
		{`'[^']*'`, `'***'`},
		// Strings entre aspas duplas
		{`"[^"]*"`, `"***"`},
		// Números que podem ser IDs ou valores sensíveis (mais de 6 dígitos)
		{`\b\d{7,}\b`, `***`},
		// Padrões de email
		{`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, `***@***.***`},
		// UUIDs
		{`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`, `***-***-***-***-***`},
	}

	result := query
	for _, p := range patterns {
		regex := pl.getCompiledRegex(p.pattern)
		if regex != nil {
			result = regex.ReplaceAllString(result, p.replacement)
		}
	}

	return result
}

// getCompiledRegex retorna um regex compilado do cache ou compila e armazena
func (pl *PgxLogger) getCompiledRegex(pattern string) *regexp.Regexp {
	queryRegexMutex.RLock()
	if regex, exists := queryRegexCache[pattern]; exists {
		queryRegexMutex.RUnlock()
		return regex
	}
	queryRegexMutex.RUnlock()

	// Compilar regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	// Armazenar no cache
	queryRegexMutex.Lock()
	queryRegexCache[pattern] = regex
	queryRegexMutex.Unlock()

	return regex
}

// WithMinLevel configura o nível mínimo de log
func (c PgxLoggerConfig) WithMinLevel(level tracelog.LogLevel) PgxLoggerConfig {
	c.MinLevel = level
	return c
}

// WithSanitizeQueries configura se queries devem ser sanitizadas
func (c PgxLoggerConfig) WithSanitizeQueries(sanitize bool) PgxLoggerConfig {
	c.SanitizeQueries = sanitize
	return c
}

// WithSanitizeArgs configura se argumentos devem ser sanitizados
func (c PgxLoggerConfig) WithSanitizeArgs(sanitize bool) PgxLoggerConfig {
	c.SanitizeArgs = sanitize
	return c
}

// WithMaxQueryLength configura o tamanho máximo da query
func (c PgxLoggerConfig) WithMaxQueryLength(length int) PgxLoggerConfig {
	c.MaxQueryLength = length
	return c
}

// WithLogger configura o logger adapter
func (c PgxLoggerConfig) WithLogger(logger core.LoggerAdapter) PgxLoggerConfig {
	c.Logger = logger
	return c
}

// ConfigurePgxPool configura um pgxpool.Config existente para usar o logger PGX
func ConfigurePgxPool(config *pgxpool.Config, loggerConfig PgxLoggerConfig) {
	pgxLogger := NewPgxLogger(loggerConfig)
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   pgxLogger,
		LogLevel: loggerConfig.MinLevel,
	}
}

// NewPgxPool cria um novo pool PGX pré-configurado com logging
func NewPgxPool(ctx context.Context, connString string, loggerConfig PgxLoggerConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	ConfigurePgxPool(config, loggerConfig)

	return pgxpool.NewWithConfig(ctx, config)
}

// NewPgxPoolWithDefaults cria um pool PGX com configurações padrão de logging
func NewPgxPoolWithDefaults(ctx context.Context, connString string, logger core.LoggerAdapter) (*pgxpool.Pool, error) {
	config := DefaultPgxLoggerConfig(logger)
	return NewPgxPool(ctx, connString, config)
}

// NewPgxPoolProduction cria um pool PGX com configurações otimizadas para produção
func NewPgxPoolProduction(ctx context.Context, connString string, logger core.LoggerAdapter) (*pgxpool.Pool, error) {
	config := DefaultPgxLoggerConfig(logger).
		WithMinLevel(tracelog.LogLevelWarn). // Apenas warnings e erros
		WithSanitizeQueries(true).           // Sempre sanitizar em produção
		WithSanitizeArgs(true).              // Sempre sanitizar argumentos
		WithMaxQueryLength(500)              // Queries menores em produção

	return NewPgxPool(ctx, connString, config)
}

// NewPgxPoolDevelopment cria um pool PGX com configurações para desenvolvimento
func NewPgxPoolDevelopment(ctx context.Context, connString string, logger core.LoggerAdapter) (*pgxpool.Pool, error) {
	config := DefaultPgxLoggerConfig(logger).
		WithMinLevel(tracelog.LogLevelDebug). // Logs detalhados para debug
		WithSanitizeQueries(false).           // Não sanitizar para debug
		WithSanitizeArgs(false).              // Mostrar argumentos para debug
		WithMaxQueryLength(2000)              // Queries maiores para análise

	return NewPgxPool(ctx, connString, config)
}

// ConfigurePgxPoolWithOptions configura um pool com opções específicas
type PgxPoolOptions struct {
	// LogLevel define o nível mínimo de log
	LogLevel tracelog.LogLevel
	// Production define se está em ambiente de produção (sanitização habilitada)
	Production bool
	// MaxQueryLength define o tamanho máximo da query para logging
	MaxQueryLength int
	// CustomConfig permite configuração personalizada adicional
	CustomConfig func(*PgxLoggerConfig)
}

// DefaultPgxPoolOptions retorna opções padrão para configuração de pool
func DefaultPgxPoolOptions() PgxPoolOptions {
	return PgxPoolOptions{
		LogLevel:       tracelog.LogLevelInfo,
		Production:     true,
		MaxQueryLength: 1000,
	}
}

// NewPgxPoolWithOptions cria um pool PGX com opções personalizadas
func NewPgxPoolWithOptions(ctx context.Context, connString string, logger core.LoggerAdapter, options PgxPoolOptions) (*pgxpool.Pool, error) {
	config := DefaultPgxLoggerConfig(logger).
		WithMinLevel(options.LogLevel).
		WithSanitizeQueries(options.Production).
		WithSanitizeArgs(options.Production).
		WithMaxQueryLength(options.MaxQueryLength)

	// Aplicar configuração personalizada se fornecida
	if options.CustomConfig != nil {
		options.CustomConfig(&config)
	}

	return NewPgxPool(ctx, connString, config)
}
