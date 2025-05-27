package core

import (
	"context"
	"testing"
)

// mockAdapter é um mock da interface LoggerAdapter para testes
type mockAdapter struct {
	logCalls       []logCall
	levelEnabled   map[Level]bool
	contextAdapter LoggerAdapter
}

type logCall struct {
	ctx    context.Context
	level  Level
	msg    string
	fields map[string]interface{}
}

func newMockAdapter() *mockAdapter {
	return &mockAdapter{
		logCalls:     make([]logCall, 0),
		levelEnabled: make(map[Level]bool),
	}
}

func (m *mockAdapter) Log(ctx context.Context, level Level, msg string, fields map[string]interface{}) {
	m.logCalls = append(m.logCalls, logCall{
		ctx:    ctx,
		level:  level,
		msg:    msg,
		fields: fields,
	})
}

func (m *mockAdapter) WithContext(ctx context.Context) LoggerAdapter {
	newAdapter := &mockAdapter{
		logCalls:       make([]logCall, 0),
		levelEnabled:   m.levelEnabled,
		contextAdapter: m,
	}
	return newAdapter
}

func (m *mockAdapter) IsLevelEnabled(level Level) bool {
	enabled, exists := m.levelEnabled[level]
	if !exists {
		return true // padrão é habilitado
	}
	return enabled
}

func (m *mockAdapter) setLevelEnabled(level Level, enabled bool) {
	m.levelEnabled[level] = enabled
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{"DEBUG level", DEBUG, "DEBUG"},
		{"INFO level", INFO, "INFO"},
		{"WARN level", WARN, "WARN"},
		{"ERROR level", ERROR, "ERROR"},
		{"FATAL level", FATAL, "FATAL"},
		{"Unknown level", Level(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Level.String() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLoggerAdapter_Log(t *testing.T) {
	ctx := context.Background()
	adapter := newMockAdapter()

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	adapter.Log(ctx, INFO, "test message", fields)

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]
	if call.ctx != ctx {
		t.Errorf("Expected context %v, got %v", ctx, call.ctx)
	}
	if call.level != INFO {
		t.Errorf("Expected level %v, got %v", INFO, call.level)
	}
	if call.msg != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", call.msg)
	}
	if len(call.fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(call.fields))
	}
	if call.fields["key1"] != "value1" {
		t.Errorf("Expected field key1='value1', got '%v'", call.fields["key1"])
	}
	if call.fields["key2"] != 42 {
		t.Errorf("Expected field key2=42, got %v", call.fields["key2"])
	}
}

func TestLoggerAdapter_WithContext(t *testing.T) {
	originalAdapter := newMockAdapter()
	ctx := context.WithValue(context.Background(), "test", "value")

	newAdapter := originalAdapter.WithContext(ctx)

	if newAdapter == originalAdapter {
		t.Error("WithContext should return a new adapter instance")
	}

	// Verificar que o novo adapter é funcional
	newAdapter.Log(ctx, INFO, "test", nil)

	// O adapter original não deve ter recebido a chamada
	if len(originalAdapter.logCalls) != 0 {
		t.Error("Original adapter should not have received log calls")
	}
}

func TestLoggerAdapter_IsLevelEnabled(t *testing.T) {
	adapter := newMockAdapter()

	// Teste padrão (todos os níveis habilitados)
	for _, level := range []Level{DEBUG, INFO, WARN, ERROR, FATAL} {
		if !adapter.IsLevelEnabled(level) {
			t.Errorf("Level %v should be enabled by default", level)
		}
	}

	// Teste com nível desabilitado
	adapter.setLevelEnabled(DEBUG, false)
	if adapter.IsLevelEnabled(DEBUG) {
		t.Error("DEBUG level should be disabled")
	}

	// Outros níveis devem continuar habilitados
	if !adapter.IsLevelEnabled(INFO) {
		t.Error("INFO level should still be enabled")
	}
}

func TestLoggerAdapter_Interface(t *testing.T) {
	// Verificar que mockAdapter implementa LoggerAdapter
	var _ LoggerAdapter = (*mockAdapter)(nil)

	// Teste de integração básica
	adapter := newMockAdapter()
	ctx := context.Background()

	// Testar sequência de operações
	adapter.Log(ctx, INFO, "first message", nil)

	newAdapter := adapter.WithContext(ctx)
	newAdapter.Log(ctx, ERROR, "second message", map[string]interface{}{"error": "test"})

	// Verificar que apenas o adapter original recebeu a primeira chamada
	if len(adapter.logCalls) != 1 {
		t.Errorf("Expected 1 log call on original adapter, got %d", len(adapter.logCalls))
	}

	if adapter.logCalls[0].msg != "first message" {
		t.Errorf("Expected 'first message', got '%s'", adapter.logCalls[0].msg)
	}
}
