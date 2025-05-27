package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/victorximenis/logger/core"
)

// ObservabilityConfig define a configuração geral de observabilidade
type ObservabilityConfig struct {
	// Enabled habilita/desabilita observabilidade
	Enabled bool
	// EnableDatadog habilita integração com Datadog
	EnableDatadog bool
	// EnableELK habilita integração com ELK
	EnableELK bool
	// EnableCorrelationID habilita geração automática de correlation IDs
	EnableCorrelationID bool
	// FallbackOnError define se deve usar fallback quando um adapter falha
	FallbackOnError bool
	// Datadog configuration
	Datadog DatadogConfig
	// ELK configuration
	ELK ELKConfig
}

// DefaultObservabilityConfig retorna configuração padrão de observabilidade
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Enabled:             getEnvBool("OBSERVABILITY_ENABLED", true),
		EnableDatadog:       getEnvBool("OBSERVABILITY_DATADOG", false),
		EnableELK:           getEnvBool("OBSERVABILITY_ELK", false),
		EnableCorrelationID: getEnvBool("OBSERVABILITY_CORRELATION_ID", true),
		FallbackOnError:     getEnvBool("OBSERVABILITY_FALLBACK", true),
		Datadog:             DefaultDatadogConfig(),
		ELK:                 DefaultELKConfig(),
	}
}

// MultiObservabilityAdapter combina múltiplos adapters de observabilidade
type MultiObservabilityAdapter struct {
	baseAdapter core.LoggerAdapter
	adapters    []core.LoggerAdapter
	config      ObservabilityConfig
	mutex       sync.RWMutex
	failedCount map[string]int
}

// NewMultiObservabilityAdapter cria um novo adapter multi-observabilidade
func NewMultiObservabilityAdapter(baseAdapter core.LoggerAdapter, config ObservabilityConfig) (*MultiObservabilityAdapter, error) {
	adapter := &MultiObservabilityAdapter{
		baseAdapter: baseAdapter,
		adapters:    make([]core.LoggerAdapter, 0),
		config:      config,
		failedCount: make(map[string]int),
	}

	if !config.Enabled {
		return adapter, nil
	}

	// Adicionar adapters baseado na configuração
	if config.EnableDatadog && config.Datadog.Enabled {
		if err := InitDatadog(config.Datadog); err != nil {
			if !config.FallbackOnError {
				return nil, fmt.Errorf("failed to initialize Datadog: %w", err)
			}
		} else {
			datadogAdapter := NewDatadogLoggerAdapter(baseAdapter, config.Datadog)
			adapter.adapters = append(adapter.adapters, datadogAdapter)
		}
	}

	if config.EnableELK && config.ELK.Enabled {
		elkAdapter := NewELKLoggerAdapter(baseAdapter, config.ELK)
		adapter.adapters = append(adapter.adapters, elkAdapter)
	}

	// Se correlation ID está habilitado, wrap com CorrelationAdapter
	if config.EnableCorrelationID {
		correlationAdapter := NewCorrelationIDAdapter(adapter, config)
		return &MultiObservabilityAdapter{
			baseAdapter: correlationAdapter,
			adapters:    []core.LoggerAdapter{correlationAdapter},
			config:      config,
			failedCount: make(map[string]int),
		}, nil
	}

	return adapter, nil
}

// Log implementa a interface LoggerAdapter
func (m *MultiObservabilityAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	if !m.config.Enabled || len(m.adapters) == 0 {
		// Se observabilidade está desabilitada ou não há adapters, usar apenas o base
		m.baseAdapter.Log(ctx, level, msg, fields)
		return
	}

	// Executar em todos os adapters
	var wg sync.WaitGroup
	for i, adapter := range m.adapters {
		wg.Add(1)
		go func(idx int, a core.LoggerAdapter) {
			defer wg.Done()
			defer m.handlePanic(idx, a)

			a.Log(ctx, level, msg, fields)
		}(i, adapter)
	}

	// Aguardar todos os adapters terminarem
	wg.Wait()
}

// handlePanic trata panics de adapters individuais
func (m *MultiObservabilityAdapter) handlePanic(idx int, adapter core.LoggerAdapter) {
	if r := recover(); r != nil {
		m.mutex.Lock()
		adapterName := fmt.Sprintf("adapter_%d", idx)
		m.failedCount[adapterName]++
		m.mutex.Unlock()

		// Log do erro usando o adapter base
		if m.baseAdapter != nil {
			m.baseAdapter.Log(context.Background(), core.ERROR, "Observability adapter panic", map[string]interface{}{
				"adapter_index": idx,
				"error":         r,
				"failed_count":  m.failedCount[adapterName],
			})
		}
	}
}

// WithContext implementa a interface LoggerAdapter
func (m *MultiObservabilityAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	newAdapters := make([]core.LoggerAdapter, len(m.adapters))
	for i, adapter := range m.adapters {
		newAdapters[i] = adapter.WithContext(ctx)
	}

	return &MultiObservabilityAdapter{
		baseAdapter: m.baseAdapter.WithContext(ctx),
		adapters:    newAdapters,
		config:      m.config,
		failedCount: m.failedCount,
	}
}

// IsLevelEnabled implementa a interface LoggerAdapter
func (m *MultiObservabilityAdapter) IsLevelEnabled(level core.Level) bool {
	return m.baseAdapter.IsLevelEnabled(level)
}

// GetFailedCounts retorna contadores de falhas por adapter
func (m *MultiObservabilityAdapter) GetFailedCounts() map[string]int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]int)
	for k, v := range m.failedCount {
		result[k] = v
	}
	return result
}

// CorrelationIDAdapter adiciona correlation IDs automaticamente
type CorrelationIDAdapter struct {
	core.LoggerAdapter
	config ObservabilityConfig
}

// NewCorrelationIDAdapter cria um novo adapter de correlation ID
func NewCorrelationIDAdapter(baseAdapter core.LoggerAdapter, config ObservabilityConfig) *CorrelationIDAdapter {
	return &CorrelationIDAdapter{
		LoggerAdapter: baseAdapter,
		config:        config,
	}
}

// Log implementa a interface LoggerAdapter com correlation ID
func (c *CorrelationIDAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	// Criar uma cópia dos campos
	enrichedFields := make(map[string]interface{})
	for k, v := range fields {
		enrichedFields[k] = v
	}

	// Adicionar correlation ID se não existir
	if _, exists := enrichedFields["correlation_id"]; !exists {
		if correlationID := c.getOrCreateCorrelationID(ctx); correlationID != "" {
			enrichedFields["correlation_id"] = correlationID
		}
	}

	// Adicionar timestamp se não existir
	if _, exists := enrichedFields["timestamp"]; !exists {
		enrichedFields["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}

	// Encaminhar para o adapter base
	c.LoggerAdapter.Log(ctx, level, msg, enrichedFields)
}

// getOrCreateCorrelationID obtém ou cria um correlation ID
func (c *CorrelationIDAdapter) getOrCreateCorrelationID(ctx context.Context) string {
	// Tentar extrair correlation ID existente do contexto
	if correlationID := c.getContextValue(ctx, "correlation_id"); correlationID != "" {
		return correlationID
	}

	// Tentar extrair de outros campos comuns
	if requestID := c.getContextValue(ctx, "request_id"); requestID != "" {
		return requestID
	}

	if traceID := c.getContextValue(ctx, "trace_id"); traceID != "" {
		return traceID
	}

	// Gerar novo correlation ID
	return uuid.New().String()
}

// getContextValue extrai um valor do contexto como string
func (c *CorrelationIDAdapter) getContextValue(ctx context.Context, key string) string {
	if value := ctx.Value(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// WithContext implementa a interface LoggerAdapter
func (c *CorrelationIDAdapter) WithContext(ctx context.Context) core.LoggerAdapter {
	return &CorrelationIDAdapter{
		LoggerAdapter: c.LoggerAdapter.WithContext(ctx),
		config:        c.config,
	}
}

// Factory functions para diferentes configurações

// NewProductionObservabilityAdapter cria adapter para produção
func NewProductionObservabilityAdapter(baseAdapter core.LoggerAdapter) (*MultiObservabilityAdapter, error) {
	config := DefaultObservabilityConfig()

	// Configurações de produção
	config.EnableDatadog = true
	config.EnableELK = true
	config.EnableCorrelationID = true
	config.FallbackOnError = true

	// Configurações específicas do Datadog para produção
	config.Datadog.Enabled = true
	config.Datadog.TracingEnabled = true
	config.Datadog.MetricsEnabled = true
	config.Datadog.SampleRate = 0.1 // 10% sampling em produção

	// Configurações específicas do ELK para produção
	config.ELK.Enabled = true
	config.ELK.EnableECSMapping = true

	return NewMultiObservabilityAdapter(baseAdapter, config)
}

// NewDevelopmentObservabilityAdapter cria adapter para desenvolvimento
func NewDevelopmentObservabilityAdapter(baseAdapter core.LoggerAdapter) (*MultiObservabilityAdapter, error) {
	config := DefaultObservabilityConfig()

	// Configurações de desenvolvimento
	config.EnableDatadog = false // Desabilitado por padrão em dev
	config.EnableELK = true
	config.EnableCorrelationID = true
	config.FallbackOnError = true

	// Configurações específicas do ELK para desenvolvimento
	config.ELK.Enabled = true
	config.ELK.EnableECSMapping = false // Formato mais simples em dev

	return NewMultiObservabilityAdapter(baseAdapter, config)
}

// NewDatadogOnlyAdapter cria adapter apenas com Datadog
func NewDatadogOnlyAdapter(baseAdapter core.LoggerAdapter, datadogConfig DatadogConfig) (*MultiObservabilityAdapter, error) {
	config := DefaultObservabilityConfig()
	config.EnableDatadog = true
	config.EnableELK = false
	config.EnableCorrelationID = true
	config.Datadog = datadogConfig

	return NewMultiObservabilityAdapter(baseAdapter, config)
}

// NewELKOnlyAdapter cria adapter apenas com ELK
func NewELKOnlyAdapter(baseAdapter core.LoggerAdapter, elkConfig ELKConfig) (*MultiObservabilityAdapter, error) {
	config := DefaultObservabilityConfig()
	config.EnableDatadog = false
	config.EnableELK = true
	config.EnableCorrelationID = true
	config.ELK = elkConfig

	return NewMultiObservabilityAdapter(baseAdapter, config)
}

// NewCustomObservabilityAdapter cria adapter com configuração personalizada
func NewCustomObservabilityAdapter(baseAdapter core.LoggerAdapter, config ObservabilityConfig) (*MultiObservabilityAdapter, error) {
	return NewMultiObservabilityAdapter(baseAdapter, config)
}

// ContextWithCorrelationID adiciona correlation ID ao contexto
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, "correlation_id", correlationID)
}

// CorrelationIDFromContext extrai correlation ID do contexto
func CorrelationIDFromContext(ctx context.Context) string {
	if value := ctx.Value("correlation_id"); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GenerateCorrelationID gera um novo correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}
