package adapters

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/victorximenis/logger/core"
)

// ZerologAdapter implementa a interface LoggerAdapter usando a biblioteca zerolog
type ZerologAdapter struct {
	logger    zerolog.Logger
	formatter *core.Formatter
}

// ZerologConfig define as opções de configuração para o ZerologAdapter
type ZerologConfig struct {
	// Writer define onde os logs serão escritos (padrão: os.Stdout)
	Writer io.Writer
	// Level define o nível mínimo de log (padrão: INFO)
	Level core.Level
	// TimeFormat define o formato do timestamp (padrão: RFC3339)
	TimeFormat string
	// PrettyPrint habilita formatação legível para desenvolvimento (padrão: false)
	PrettyPrint bool
	// CallerEnabled habilita informações do caller nos logs (padrão: false)
	CallerEnabled bool
	// FormatterConfig define a configuração para o formatter JSON
	FormatterConfig *core.Config
}

// NewZerologAdapter cria uma nova instância do ZerologAdapter com a configuração especificada.
// Se config for nil, usa configurações padrão adequadas para produção.
func NewZerologAdapter(config *ZerologConfig) *ZerologAdapter {
	if config == nil {
		config = &ZerologConfig{
			Writer:        os.Stdout,
			Level:         core.INFO,
			TimeFormat:    zerolog.TimeFormatUnix,
			PrettyPrint:   false,
			CallerEnabled: false,
			FormatterConfig: &core.Config{
				ServiceName:           "unknown-service",
				Environment:           "development",
				TenantID:              "",
				SanitizeSensitiveData: false,
			},
		}
	}

	// Configurar writer
	writer := config.Writer
	if writer == nil {
		writer = os.Stdout
	}

	// Configurar pretty print para desenvolvimento
	if config.PrettyPrint {
		writer = zerolog.ConsoleWriter{Out: writer}
	}

	// Criar logger base
	logger := zerolog.New(writer)

	// Configurar timestamp
	if config.TimeFormat != "" {
		zerolog.TimeFieldFormat = config.TimeFormat
		logger = logger.With().Timestamp().Logger()
	}

	// Configurar caller se habilitado
	if config.CallerEnabled {
		logger = logger.With().Caller().Logger()
	}

	// Configurar nível de log
	logger = logger.Level(mapLevelToZerolog(config.Level))

	// Criar formatter
	var formatter *core.Formatter
	if config.FormatterConfig != nil {
		formatter = core.NewFormatter(*config.FormatterConfig)
	} else {
		formatter = core.NewFormatter(core.Config{
			ServiceName:           "unknown-service",
			Environment:           "development",
			TenantID:              "",
			SanitizeSensitiveData: false,
		})
	}

	return &ZerologAdapter{
		logger:    logger,
		formatter: formatter,
	}
}

// NewZerologAdapterFromLogger cria um ZerologAdapter a partir de um zerolog.Logger existente.
// Útil quando você já tem um logger zerolog configurado e quer usar com a interface unificada.
func NewZerologAdapterFromLogger(logger zerolog.Logger) *ZerologAdapter {
	// Usar configuração padrão para o formatter
	formatter := core.NewFormatter(core.Config{
		ServiceName:           "unknown-service",
		Environment:           "development",
		TenantID:              "",
		SanitizeSensitiveData: false,
	})

	return &ZerologAdapter{
		logger:    logger,
		formatter: formatter,
	}
}

// NewZerologAdapterFromLoggerWithFormatter cria um ZerologAdapter a partir de um zerolog.Logger
// existente e um formatter customizado.
func NewZerologAdapterFromLoggerWithFormatter(logger zerolog.Logger, formatter *core.Formatter) *ZerologAdapter {
	return &ZerologAdapter{
		logger:    logger,
		formatter: formatter,
	}
}

// Log implementa o método Log da interface LoggerAdapter
func (z *ZerologAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	if !z.IsLevelEnabled(level) {
		return
	}

	// Usar formatter para padronizar os campos do log
	formattedFields := z.formatter.FormatLogEvent(ctx, level, msg, fields)

	// Criar evento de log com o nível apropriado
	var event *zerolog.Event
	switch level {
	case core.DEBUG:
		event = z.logger.Debug()
	case core.INFO:
		event = z.logger.Info()
	case core.WARN:
		event = z.logger.Warn()
	case core.ERROR:
		event = z.logger.Error()
	case core.FATAL:
		event = z.logger.Fatal()
	default:
		event = z.logger.Info()
	}

	// Adicionar contexto se disponível
	if ctx != nil {
		event = event.Ctx(ctx)
	}

	// Adicionar todos os campos formatados
	for key, value := range formattedFields {
		if key != "message" { // Mensagem é tratada separadamente
			event = addFieldToEvent(event, key, value)
		}
	}

	// Enviar mensagem
	event.Msg(msg)
}

// WithContext implementa o método WithContext da interface LoggerAdapter
func (z *ZerologAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	// Criar novo logger com contexto
	newLogger := z.logger.With().Logger()

	// Se o contexto não for nil, criar um logger que usará esse contexto
	if ctx != nil {
		newLogger = newLogger.With().Logger()
	}

	return &ZerologAdapter{
		logger:    newLogger,
		formatter: z.formatter, // Preservar o formatter
	}
}

// IsLevelEnabled implementa o método IsLevelEnabled da interface LoggerAdapter
func (z *ZerologAdapter) IsLevelEnabled(level core.Level) bool {
	zerologLevel := mapLevelToZerolog(level)
	return z.logger.GetLevel() <= zerologLevel
}

// mapLevelToZerolog mapeia os níveis customizados para os níveis do zerolog
func mapLevelToZerolog(level core.Level) zerolog.Level {
	switch level {
	case core.DEBUG:
		return zerolog.DebugLevel
	case core.INFO:
		return zerolog.InfoLevel
	case core.WARN:
		return zerolog.WarnLevel
	case core.ERROR:
		return zerolog.ErrorLevel
	case core.FATAL:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// addFieldToEvent adiciona um campo ao evento zerolog, tratando tipos especiais
func addFieldToEvent(event *zerolog.Event, key string, value interface{}) *zerolog.Event {
	switch v := value.(type) {
	case string:
		return event.Str(key, v)
	case int:
		return event.Int(key, v)
	case int32:
		return event.Int32(key, v)
	case int64:
		return event.Int64(key, v)
	case float32:
		return event.Float32(key, v)
	case float64:
		return event.Float64(key, v)
	case bool:
		return event.Bool(key, v)
	case error:
		return event.AnErr(key, v)
	default:
		return event.Interface(key, v)
	}
}
