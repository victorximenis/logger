package observability

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/victorximenis/logger/core"
)

// ELKConfig contém a configuração para integração com ELK Stack
type ELKConfig struct {
	// Enabled habilita/desabilita a integração com ELK
	Enabled bool
	// IndexPrefix define o prefixo dos índices no Elasticsearch
	IndexPrefix string
	// Environment define o ambiente (dev, staging, prod)
	Environment string
	// ServiceName define o nome do serviço
	ServiceName string
	// ServiceVersion define a versão do serviço
	ServiceVersion string
	// DatacenterName define o nome do datacenter
	DatacenterName string
	// HostName define o nome do host
	HostName string
	// EnableECSMapping habilita mapeamento para Elastic Common Schema
	EnableECSMapping bool
	// CustomFields define campos personalizados para adicionar a todos os logs
	CustomFields map[string]interface{}
}

// DefaultELKConfig retorna a configuração padrão do ELK
func DefaultELKConfig() ELKConfig {
	hostname, _ := os.Hostname()

	return ELKConfig{
		Enabled:          getEnvBool("ELK_ENABLED", false),
		IndexPrefix:      getEnvOrDefault("ELK_INDEX_PREFIX", "logs"),
		Environment:      getEnvOrDefault("ELK_ENV", "development"),
		ServiceName:      getEnvOrDefault("ELK_SERVICE", "unknown-service"),
		ServiceVersion:   getEnvOrDefault("ELK_SERVICE_VERSION", "1.0.0"),
		DatacenterName:   getEnvOrDefault("ELK_DATACENTER", "local"),
		HostName:         getEnvOrDefault("ELK_HOSTNAME", hostname),
		EnableECSMapping: getEnvBool("ELK_ECS_MAPPING", true),
		CustomFields:     parseCustomFields("ELK_CUSTOM_FIELDS"),
	}
}

// ELKLoggerAdapter aprimora o logger com funcionalidades específicas do ELK
type ELKLoggerAdapter struct {
	core.LoggerAdapter
	config ELKConfig
}

// NewELKLoggerAdapter cria um novo adapter de logger aprimorado com ELK
func NewELKLoggerAdapter(baseAdapter core.LoggerAdapter, config ELKConfig) *ELKLoggerAdapter {
	return &ELKLoggerAdapter{
		LoggerAdapter: baseAdapter,
		config:        config,
	}
}

// Log implementa a interface LoggerAdapter com melhorias do ELK
func (e *ELKLoggerAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	// Criar uma cópia dos campos para não modificar o original
	enrichedFields := make(map[string]interface{})
	for k, v := range fields {
		enrichedFields[k] = v
	}

	// Adicionar campos personalizados
	for k, v := range e.config.CustomFields {
		enrichedFields[k] = v
	}

	if e.config.EnableECSMapping {
		e.applyECSMapping(ctx, level, msg, enrichedFields)
	} else {
		e.applyBasicMapping(ctx, level, msg, enrichedFields)
	}

	// Encaminhar para o adapter base
	e.LoggerAdapter.Log(ctx, level, msg, enrichedFields)
}

// applyECSMapping aplica mapeamento para Elastic Common Schema (ECS)
func (e *ELKLoggerAdapter) applyECSMapping(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	// Timestamp no formato ECS
	if _, exists := fields["@timestamp"]; !exists {
		fields["@timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	// Campos básicos ECS
	fields["ecs.version"] = "8.0"
	fields["message"] = msg

	// Log level mapping
	fields["log.level"] = strings.ToLower(level.String())
	fields["log.logger"] = "application"

	// Service information
	if e.config.ServiceName != "" {
		fields["service.name"] = e.config.ServiceName
	}
	if e.config.ServiceVersion != "" {
		fields["service.version"] = e.config.ServiceVersion
	}
	if e.config.Environment != "" {
		fields["service.environment"] = e.config.Environment
	}

	// Host information
	if e.config.HostName != "" {
		fields["host.name"] = e.config.HostName
	}
	if e.config.DatacenterName != "" {
		fields["cloud.availability_zone"] = e.config.DatacenterName
	}

	// Process information
	fields["process.pid"] = os.Getpid()

	// Extrair informações do contexto
	e.extractContextFields(ctx, fields)

	// Mapear campos existentes para ECS
	e.mapExistingFieldsToECS(fields)
}

// applyBasicMapping aplica mapeamento básico sem ECS
func (e *ELKLoggerAdapter) applyBasicMapping(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	// Timestamp básico
	if _, exists := fields["timestamp"]; !exists {
		fields["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	// Campos básicos
	fields["level"] = level.String()
	fields["message"] = msg
	fields["service"] = e.config.ServiceName
	fields["environment"] = e.config.Environment
	fields["version"] = e.config.ServiceVersion
	fields["hostname"] = e.config.HostName

	// Extrair informações do contexto
	e.extractContextFields(ctx, fields)
}

// extractContextFields extrai campos do contexto
func (e *ELKLoggerAdapter) extractContextFields(ctx context.Context, fields map[string]interface{}) {
	// Extrair user ID do contexto
	if userID := e.getContextValue(ctx, "user_id"); userID != "" {
		if e.config.EnableECSMapping {
			fields["user.id"] = userID
		} else {
			fields["user_id"] = userID
		}
	}

	// Extrair trace ID do contexto
	if traceID := e.getContextValue(ctx, "trace_id"); traceID != "" {
		if e.config.EnableECSMapping {
			fields["trace.id"] = traceID
		} else {
			fields["trace_id"] = traceID
		}
	}

	// Extrair span ID do contexto
	if spanID := e.getContextValue(ctx, "span_id"); spanID != "" {
		if e.config.EnableECSMapping {
			fields["span.id"] = spanID
		} else {
			fields["span_id"] = spanID
		}
	}

	// Extrair request ID do contexto
	if requestID := e.getContextValue(ctx, "request_id"); requestID != "" {
		if e.config.EnableECSMapping {
			fields["http.request.id"] = requestID
		} else {
			fields["request_id"] = requestID
		}
	}

	// Extrair correlation ID do contexto
	if correlationID := e.getContextValue(ctx, "correlation_id"); correlationID != "" {
		if e.config.EnableECSMapping {
			fields["labels.correlation_id"] = correlationID
		} else {
			fields["correlation_id"] = correlationID
		}
	}

	// Extrair session ID do contexto
	if sessionID := e.getContextValue(ctx, "session_id"); sessionID != "" {
		if e.config.EnableECSMapping {
			fields["user.session.id"] = sessionID
		} else {
			fields["session_id"] = sessionID
		}
	}
}

// mapExistingFieldsToECS mapeia campos existentes para ECS
func (e *ELKLoggerAdapter) mapExistingFieldsToECS(fields map[string]interface{}) {
	// Mapear campos de erro
	if err, exists := fields["error"]; exists {
		fields["error.message"] = err
		delete(fields, "error")
	}

	// Mapear campos de duração
	if duration, exists := fields["duration"]; exists {
		fields["event.duration"] = duration
	}
	if durationMs, exists := fields["duration_ms"]; exists {
		// Converter milliseconds para nanoseconds (ECS usa nanoseconds)
		if ms, ok := durationMs.(float64); ok {
			fields["event.duration"] = int64(ms * 1000000) // ms to ns
		} else if ms, ok := durationMs.(int64); ok {
			fields["event.duration"] = ms * 1000000 // ms to ns
		}
		delete(fields, "duration_ms")
	}

	// Mapear campos HTTP
	if method, exists := fields["method"]; exists {
		fields["http.request.method"] = method
		delete(fields, "method")
	}
	if path, exists := fields["path"]; exists {
		fields["url.path"] = path
		delete(fields, "path")
	}
	if statusCode, exists := fields["status_code"]; exists {
		fields["http.response.status_code"] = statusCode
		delete(fields, "status_code")
	}
	if userAgent, exists := fields["user_agent"]; exists {
		fields["user_agent.original"] = userAgent
		delete(fields, "user_agent")
	}
	if remoteIP, exists := fields["remote_ip"]; exists {
		fields["client.ip"] = remoteIP
		delete(fields, "remote_ip")
	}

	// Mapear campos de componente
	if component, exists := fields["component"]; exists {
		fields["labels.component"] = component
		delete(fields, "component")
	}
}

// getContextValue extrai um valor do contexto como string
func (e *ELKLoggerAdapter) getContextValue(ctx context.Context, key string) string {
	if value := ctx.Value(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// WithContext implementa a interface LoggerAdapter
func (e *ELKLoggerAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	return &ELKLoggerAdapter{
		LoggerAdapter: e.LoggerAdapter.WithContext(ctx),
		config:        e.config,
	}
}

// IsLevelEnabled implementa a interface LoggerAdapter
func (e *ELKLoggerAdapter) IsLevelEnabled(level core.Level) bool {
	return e.LoggerAdapter.IsLevelEnabled(level)
}

// Funções auxiliares

// parseCustomFields parseia campos personalizados da variável de ambiente
func parseCustomFields(envKey string) map[string]interface{} {
	value := os.Getenv(envKey)
	if value == "" {
		return make(map[string]interface{})
	}

	fields := make(map[string]interface{})
	// Formato esperado: "key1=value1,key2=value2"
	pairs := splitAndTrim(value, ",")
	for _, pair := range pairs {
		if pair != "" {
			parts := splitAndTrim(pair, "=")
			if len(parts) == 2 {
				fields[parts[0]] = parts[1]
			}
		}
	}
	return fields
}

// GetECSIndexName retorna o nome do índice ECS baseado na configuração
func (e *ELKLoggerAdapter) GetECSIndexName() string {
	if e.config.IndexPrefix == "" {
		return "logs-" + e.config.ServiceName + "-" + time.Now().Format("2006.01.02")
	}
	return e.config.IndexPrefix + "-" + e.config.ServiceName + "-" + time.Now().Format("2006.01.02")
}

// GetECSTemplate retorna um template básico para Elasticsearch
func GetECSTemplate() map[string]interface{} {
	return map[string]interface{}{
		"index_patterns": []string{"logs-*"},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":       1,
				"number_of_replicas":     0,
				"index.refresh_interval": "5s",
			},
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"@timestamp": map[string]interface{}{
						"type": "date",
					},
					"message": map[string]interface{}{
						"type": "text",
					},
					"log.level": map[string]interface{}{
						"type": "keyword",
					},
					"service.name": map[string]interface{}{
						"type": "keyword",
					},
					"service.version": map[string]interface{}{
						"type": "keyword",
					},
					"service.environment": map[string]interface{}{
						"type": "keyword",
					},
					"host.name": map[string]interface{}{
						"type": "keyword",
					},
					"user.id": map[string]interface{}{
						"type": "keyword",
					},
					"trace.id": map[string]interface{}{
						"type": "keyword",
					},
					"span.id": map[string]interface{}{
						"type": "keyword",
					},
					"http.request.method": map[string]interface{}{
						"type": "keyword",
					},
					"http.response.status_code": map[string]interface{}{
						"type": "integer",
					},
					"url.path": map[string]interface{}{
						"type": "keyword",
					},
					"client.ip": map[string]interface{}{
						"type": "ip",
					},
					"event.duration": map[string]interface{}{
						"type": "long",
					},
					"error.message": map[string]interface{}{
						"type": "text",
					},
				},
			},
		},
	}
}
