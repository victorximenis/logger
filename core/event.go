package core

import (
	"context"
	"fmt"
)

// LogEvent define a interface para construção fluente de entradas de log
// através do padrão method chaining. Permite adicionar campos e metadados
// de forma encadeada antes de enviar a mensagem de log.
type LogEvent interface {
	// Str adiciona um campo string à entrada de log
	Str(key, val string) LogEvent

	// Int adiciona um campo inteiro à entrada de log
	Int(key string, val int) LogEvent

	// Float64 adiciona um campo float64 à entrada de log
	Float64(key string, val float64) LogEvent

	// Bool adiciona um campo booleano à entrada de log
	Bool(key string, val bool) LogEvent

	// Err adiciona um erro à entrada de log com a chave "error"
	Err(err error) LogEvent

	// Any adiciona um campo de qualquer tipo à entrada de log
	Any(key string, val interface{}) LogEvent

	// Fields adiciona múltiplos campos de uma vez à entrada de log
	Fields(fields map[string]interface{}) LogEvent

	// Msg finaliza a construção da entrada de log e a envia com a mensagem especificada
	Msg(msg string)

	// Msgf finaliza a construção da entrada de log e a envia com uma mensagem formatada
	Msgf(format string, args ...interface{})

	// Send finaliza a construção da entrada de log e a envia sem mensagem adicional
	Send()
}

// logEvent é a implementação concreta da interface LogEvent
type logEvent struct {
	adapter LoggerAdapter
	ctx     context.Context
	level   Level
	fields  map[string]interface{}
}

// NewLogEvent cria uma nova instância de LogEvent
func NewLogEvent(adapter LoggerAdapter, ctx context.Context, level Level) LogEvent {
	return &logEvent{
		adapter: adapter,
		ctx:     ctx,
		level:   level,
		fields:  make(map[string]interface{}),
	}
}

// Str adiciona um campo string à entrada de log
func (e *logEvent) Str(key, val string) LogEvent {
	e.fields[key] = val
	return e
}

// Int adiciona um campo inteiro à entrada de log
func (e *logEvent) Int(key string, val int) LogEvent {
	e.fields[key] = val
	return e
}

// Float64 adiciona um campo float64 à entrada de log
func (e *logEvent) Float64(key string, val float64) LogEvent {
	e.fields[key] = val
	return e
}

// Bool adiciona um campo booleano à entrada de log
func (e *logEvent) Bool(key string, val bool) LogEvent {
	e.fields[key] = val
	return e
}

// Err adiciona um erro à entrada de log
func (e *logEvent) Err(err error) LogEvent {
	if err != nil {
		e.fields["error"] = err.Error()
	}
	return e
}

// Any adiciona um campo de qualquer tipo à entrada de log
func (e *logEvent) Any(key string, val interface{}) LogEvent {
	e.fields[key] = val
	return e
}

// Fields adiciona múltiplos campos de uma vez à entrada de log
func (e *logEvent) Fields(fields map[string]interface{}) LogEvent {
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

// Msg finaliza a construção da entrada de log e a envia
func (e *logEvent) Msg(msg string) {
	if e.adapter.IsLevelEnabled(e.level) {
		e.adapter.Log(e.ctx, e.level, msg, e.fields)
	}
}

// Msgf finaliza a construção da entrada de log e a envia com formatação
func (e *logEvent) Msgf(format string, args ...interface{}) {
	if e.adapter.IsLevelEnabled(e.level) {
		msg := fmt.Sprintf(format, args...)
		e.adapter.Log(e.ctx, e.level, msg, e.fields)
	}
}

// Send finaliza a construção da entrada de log e a envia sem mensagem
func (e *logEvent) Send() {
	if e.adapter.IsLevelEnabled(e.level) {
		e.adapter.Log(e.ctx, e.level, "", e.fields)
	}
}
