package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/victorximenis/logger/sanitize"
)

// Config representa a configuração necessária para o formatter
type Config struct {
	ServiceName           string
	Environment           string
	TenantID              string
	SanitizeSensitiveData bool
}

// Formatter é responsável por formatar eventos de log em estruturas JSON padronizadas
type Formatter struct {
	config Config
}

// NewFormatter cria uma nova instância do Formatter com a configuração especificada
func NewFormatter(config Config) *Formatter {
	return &Formatter{config: config}
}

// FormatLogEvent formata um evento de log em uma estrutura JSON padronizada
// incluindo campos base, enriquecimento de contexto e campos customizados
func (f *Formatter) FormatLogEvent(ctx context.Context, level Level, msg string, fields map[string]interface{}) map[string]interface{} {
	// Começar com campos base
	result := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"service":   f.config.ServiceName,
		"env":       f.config.Environment,
		"message":   msg,
	}

	// Adicionar tenant se disponível
	if f.config.TenantID != "" {
		result["tenant"] = f.config.TenantID
	}

	// Adicionar valores do contexto
	result = f.enrichFromContext(ctx, result)

	// Adicionar campos customizados
	for k, v := range fields {
		result[k] = v
	}

	// Sanitizar campos sensíveis se habilitado
	if f.config.SanitizeSensitiveData {
		result = f.sanitizeFields(result)
	}

	return result
}

// enrichFromContext extrai valores do contexto e os adiciona aos campos do log
func (f *Formatter) enrichFromContext(ctx context.Context, fields map[string]interface{}) map[string]interface{} {
	// Extrair e adicionar trace ID se presente
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields["trace_id"] = traceID
	}

	// Extrair e adicionar correlation ID se presente
	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok && correlationID != "" {
		fields["correlation_id"] = correlationID
	}

	// Extrair e adicionar user ID se presente
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		fields["user_id"] = userID
	}

	return fields
}

// sanitizeFields aplica sanitização aos campos do log usando as regras padrão
func (f *Formatter) sanitizeFields(fields map[string]interface{}) map[string]interface{} {
	sanitizeConfig := sanitize.DefaultSensitiveFieldConfig()

	// Converter para JSON e sanitizar
	if jsonData, err := json.Marshal(fields); err == nil {
		if sanitizedData, err := sanitize.SanitizeJSON(jsonData, sanitizeConfig); err == nil {
			var sanitizedFields map[string]interface{}
			if err := json.Unmarshal(sanitizedData, &sanitizedFields); err == nil {
				return sanitizedFields
			}
		}
	}

	// Fallback: sanitizar strings individuais se a sanitização JSON falhar
	result := make(map[string]interface{})
	for k, v := range fields {
		if str, ok := v.(string); ok {
			result[k] = sanitize.SanitizeString(str, sanitizeConfig)
		} else {
			result[k] = v
		}
	}

	return result
}
