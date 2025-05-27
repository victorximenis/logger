package logger

import (
	"context"

	"github.com/victorximenis/logger/core"
)

// Logger define a interface pública para operações de logging.
// Esta interface fornece métodos para diferentes níveis de log
// e suporta method chaining através da interface LogEvent.
//
// Exemplo de uso:
//
//	logger.Info(ctx).
//		Str("user_id", "123").
//		Int("attempt", 1).
//		Msg("User login successful")
type Logger interface {
	// Debug cria uma entrada de log de nível DEBUG.
	// Usado para informações detalhadas de depuração que normalmente
	// só são de interesse ao diagnosticar problemas.
	Debug(ctx context.Context) core.LogEvent

	// Info cria uma entrada de log de nível INFO.
	// Usado para mensagens informativas gerais que destacam
	// o progresso da aplicação em um nível grosso.
	Info(ctx context.Context) core.LogEvent

	// Warn cria uma entrada de log de nível WARN.
	// Usado para situações potencialmente prejudiciais que
	// merecem atenção mas não impedem a execução.
	Warn(ctx context.Context) core.LogEvent

	// Error cria uma entrada de log de nível ERROR.
	// Usado para eventos de erro que ainda permitem que
	// a aplicação continue executando.
	Error(ctx context.Context) core.LogEvent

	// Fatal cria uma entrada de log de nível FATAL.
	// Usado para erros muito severos que provavelmente
	// levarão à terminação da aplicação.
	Fatal(ctx context.Context) core.LogEvent

	// WithContext retorna uma nova instância do logger com o contexto especificado.
	// Útil para propagar informações de contexto através de chamadas de log.
	WithContext(ctx context.Context) Logger

	// WithFields retorna uma nova instância do logger com campos pré-definidos.
	// Útil para adicionar campos comuns que serão incluídos em todas as entradas de log.
	WithFields(fields map[string]interface{}) Logger
}

// logger é a implementação concreta da interface Logger
type logger struct {
	adapter LoggerAdapter
	ctx     context.Context
	fields  map[string]interface{}
}

// LoggerAdapter é um alias para core.LoggerAdapter para facilitar o uso
type LoggerAdapter = core.LoggerAdapter

// New cria uma nova instância de Logger usando o adapter especificado.
// O adapter é responsável pela implementação real do logging.
//
// Exemplo:
//
//	adapter := &MyLoggerAdapter{}
//	log := logger.New(adapter)
func New(adapter LoggerAdapter) Logger {
	return &logger{
		adapter: adapter,
		ctx:     context.Background(),
		fields:  make(map[string]interface{}),
	}
}

// Debug cria uma entrada de log de nível DEBUG
func (l *logger) Debug(ctx context.Context) core.LogEvent {
	event := core.NewLogEvent(l.adapter, ctx, core.DEBUG)
	return l.addPresetFields(event)
}

// Info cria uma entrada de log de nível INFO
func (l *logger) Info(ctx context.Context) core.LogEvent {
	event := core.NewLogEvent(l.adapter, ctx, core.INFO)
	return l.addPresetFields(event)
}

// Warn cria uma entrada de log de nível WARN
func (l *logger) Warn(ctx context.Context) core.LogEvent {
	event := core.NewLogEvent(l.adapter, ctx, core.WARN)
	return l.addPresetFields(event)
}

// Error cria uma entrada de log de nível ERROR
func (l *logger) Error(ctx context.Context) core.LogEvent {
	event := core.NewLogEvent(l.adapter, ctx, core.ERROR)
	return l.addPresetFields(event)
}

// Fatal cria uma entrada de log de nível FATAL
func (l *logger) Fatal(ctx context.Context) core.LogEvent {
	event := core.NewLogEvent(l.adapter, ctx, core.FATAL)
	return l.addPresetFields(event)
}

// WithContext retorna uma nova instância do logger com o contexto especificado
func (l *logger) WithContext(ctx context.Context) Logger {
	return &logger{
		adapter: l.adapter.WithContext(ctx),
		ctx:     ctx,
		fields:  l.copyFields(),
	}
}

// WithFields retorna uma nova instância do logger com campos pré-definidos
func (l *logger) WithFields(fields map[string]interface{}) Logger {
	newFields := l.copyFields()
	for k, v := range fields {
		newFields[k] = v
	}

	return &logger{
		adapter: l.adapter,
		ctx:     l.ctx,
		fields:  newFields,
	}
}

// addPresetFields adiciona os campos pré-definidos ao evento de log
func (l *logger) addPresetFields(event core.LogEvent) core.LogEvent {
	if len(l.fields) > 0 {
		event = event.Fields(l.fields)
	}
	return event
}

// copyFields cria uma cópia dos campos para evitar modificações acidentais
func (l *logger) copyFields() map[string]interface{} {
	fields := make(map[string]interface{}, len(l.fields))
	for k, v := range l.fields {
		fields[k] = v
	}
	return fields
}
