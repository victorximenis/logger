package observability

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/victorximenis/logger/core"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// DatadogConfig contém a configuração para integração com Datadog
type DatadogConfig struct {
	// Enabled habilita/desabilita a integração com Datadog
	Enabled bool
	// AgentHost define o endereço do agente Datadog
	AgentHost string
	// ServiceName define o nome do serviço
	ServiceName string
	// Environment define o ambiente (dev, staging, prod)
	Environment string
	// Version define a versão da aplicação
	Version string
	// TracingEnabled habilita/desabilita distributed tracing
	TracingEnabled bool
	// MetricsEnabled habilita/desabilita métricas
	MetricsEnabled bool
	// SampleRate define a taxa de amostragem para traces (0.0 a 1.0)
	SampleRate float64
	// Tags globais para adicionar a todos os logs/métricas
	GlobalTags []string
}

// DefaultDatadogConfig retorna a configuração padrão do Datadog
func DefaultDatadogConfig() DatadogConfig {
	sampleRate := 1.0
	if rate := os.Getenv("DD_TRACE_SAMPLE_RATE"); rate != "" {
		if parsed, err := strconv.ParseFloat(rate, 64); err == nil {
			sampleRate = parsed
		}
	}

	return DatadogConfig{
		Enabled:        getEnvBool("DD_ENABLED", false),
		AgentHost:      getEnvOrDefault("DD_AGENT_HOST", "localhost:8126"),
		ServiceName:    getEnvOrDefault("DD_SERVICE", "unknown-service"),
		Environment:    getEnvOrDefault("DD_ENV", "development"),
		Version:        getEnvOrDefault("DD_VERSION", "1.0.0"),
		TracingEnabled: getEnvBool("DD_TRACING_ENABLED", true),
		MetricsEnabled: getEnvBool("DD_METRICS_ENABLED", true),
		SampleRate:     sampleRate,
		GlobalTags:     parseEnvTags("DD_TAGS"),
	}
}

// InitDatadog inicializa a integração com Datadog
func InitDatadog(config DatadogConfig) error {
	if !config.Enabled {
		return nil
	}

	// Inicializar tracer do Datadog se habilitado
	if config.TracingEnabled {
		tracer.Start(
			tracer.WithAgentAddr(config.AgentHost),
			tracer.WithService(config.ServiceName),
			tracer.WithEnv(config.Environment),
			tracer.WithServiceVersion(config.Version),
			tracer.WithSampler(tracer.NewRateSampler(config.SampleRate)),
			tracer.WithGlobalTag("service", config.ServiceName),
			tracer.WithGlobalTag("env", config.Environment),
			tracer.WithGlobalTag("version", config.Version),
		)
	}

	// Inicializar cliente de métricas do Datadog se habilitado
	if config.MetricsEnabled {
		// Configurar opções do cliente statsd
		options := []statsd.Option{
			statsd.WithNamespace(config.ServiceName + "."),
		}

		// Adicionar tags globais
		globalTags := []string{
			"env:" + config.Environment,
			"version:" + config.Version,
		}
		globalTags = append(globalTags, config.GlobalTags...)
		if len(globalTags) > 0 {
			options = append(options, statsd.WithTags(globalTags))
		}

		client, err := statsd.New(config.AgentHost, options...)
		if err != nil {
			return err
		}

		// Armazenar o cliente para uso posterior
		datadogClient = client
	}

	return nil
}

// StopDatadog para a integração com Datadog
func StopDatadog() {
	if datadogClient != nil {
		datadogClient.Close()
		datadogClient = nil
	}
	tracer.Stop()
}

// DatadogLoggerAdapter aprimora o logger com funcionalidades específicas do Datadog
type DatadogLoggerAdapter struct {
	core.LoggerAdapter
	config DatadogConfig
}

// NewDatadogLoggerAdapter cria um novo adapter de logger aprimorado com Datadog
func NewDatadogLoggerAdapter(baseAdapter core.LoggerAdapter, config DatadogConfig) *DatadogLoggerAdapter {
	return &DatadogLoggerAdapter{
		LoggerAdapter: baseAdapter,
		config:        config,
	}
}

// Log implementa a interface LoggerAdapter com melhorias do Datadog
func (d *DatadogLoggerAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	// Extrair trace e span IDs do contexto se disponível
	if d.config.TracingEnabled {
		if span, ok := tracer.SpanFromContext(ctx); ok {
			spanContext := span.Context()
			fields["dd.trace_id"] = spanContext.TraceID()
			fields["dd.span_id"] = spanContext.SpanID()
		}
	}

	// Adicionar tags específicas do Datadog
	fields["dd.service"] = d.config.ServiceName
	fields["dd.env"] = d.config.Environment
	fields["dd.version"] = d.config.Version

	// Adicionar timestamp no formato esperado pelo Datadog
	if _, exists := fields["timestamp"]; !exists {
		fields["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	// Enviar métricas baseadas no nível de log
	if d.config.MetricsEnabled && datadogClient != nil {
		tags := []string{
			"level:" + level.String(),
			"service:" + d.config.ServiceName,
			"env:" + d.config.Environment,
		}

		// Incrementar contador de logs por nível
		datadogClient.Incr("logger.log_count", tags, 1)

		// Incrementar contador de erros se for nível ERROR ou FATAL
		if level >= core.ERROR {
			datadogClient.Incr("logger.error_count", tags, 1)
		}
	}

	// Encaminhar para o adapter base
	d.LoggerAdapter.Log(ctx, level, msg, fields)
}

// WithContext implementa a interface LoggerAdapter
func (d *DatadogLoggerAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	return &DatadogLoggerAdapter{
		LoggerAdapter: d.LoggerAdapter.WithContext(ctx),
		config:        d.config,
	}
}

// IsLevelEnabled implementa a interface LoggerAdapter
func (d *DatadogLoggerAdapter) IsLevelEnabled(level core.Level) bool {
	return d.LoggerAdapter.IsLevelEnabled(level)
}

// Cliente global do Datadog para métricas
var datadogClient *statsd.Client

// IncrementCounter incrementa um contador no Datadog
func IncrementCounter(name string, tags []string) {
	if datadogClient != nil {
		datadogClient.Incr(name, tags, 1)
	}
}

// RecordDuration registra uma duração no Datadog
func RecordDuration(name string, duration time.Duration, tags []string) {
	if datadogClient != nil {
		datadogClient.Timing(name, duration, tags, 1)
	}
}

// RecordGauge registra um valor gauge no Datadog
func RecordGauge(name string, value float64, tags []string) {
	if datadogClient != nil {
		datadogClient.Gauge(name, value, tags, 1)
	}
}

// RecordHistogram registra um valor histogram no Datadog
func RecordHistogram(name string, value float64, tags []string) {
	if datadogClient != nil {
		datadogClient.Histogram(name, value, tags, 1)
	}
}

// StartSpan inicia um novo span do Datadog
func StartSpan(operationName string, opts ...tracer.StartSpanOption) tracer.Span {
	return tracer.StartSpan(operationName, opts...)
}

// SpanFromContext extrai um span do contexto
func SpanFromContext(ctx context.Context) (tracer.Span, bool) {
	return tracer.SpanFromContext(ctx)
}

// ContextWithSpan adiciona um span ao contexto
func ContextWithSpan(ctx context.Context, span tracer.Span) context.Context {
	return tracer.ContextWithSpan(ctx, span)
}

// Funções auxiliares

// getEnvOrDefault retorna o valor da variável de ambiente ou o valor padrão
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool retorna o valor booleano da variável de ambiente ou o valor padrão
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// parseEnvTags parseia tags da variável de ambiente DD_TAGS
func parseEnvTags(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}

	// Formato esperado: "key1:value1,key2:value2"
	var tags []string
	pairs := splitAndTrim(value, ",")
	for _, pair := range pairs {
		if pair != "" {
			tags = append(tags, pair)
		}
	}
	return tags
}

// splitAndTrim divide uma string e remove espaços em branco
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString divide uma string pelo separador
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}

	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace remove espaços em branco do início e fim da string
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Remove espaços do início
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Remove espaços do fim
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
