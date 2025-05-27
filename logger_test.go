package logger

import (
	"context"
	"testing"

	"github.com/victorximenis/logger/core"
)

func TestNew(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)

	if logger == nil {
		t.Fatal("New should not return nil")
	}

	// Verificar que implementa a interface Logger
	var _ Logger = logger
}

func TestLogger_Debug(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	event := logger.Debug(ctx)

	if event == nil {
		t.Fatal("Debug should not return nil")
	}

	// Verificar que implementa a interface LogEvent
	var _ core.LogEvent = event
}

func TestLogger_Info(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	event := logger.Info(ctx)

	if event == nil {
		t.Fatal("Info should not return nil")
	}

	var _ core.LogEvent = event
}

func TestLogger_Warn(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	event := logger.Warn(ctx)

	if event == nil {
		t.Fatal("Warn should not return nil")
	}

	var _ core.LogEvent = event
}

func TestLogger_Error(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	event := logger.Error(ctx)

	if event == nil {
		t.Fatal("Error should not return nil")
	}

	var _ core.LogEvent = event
}

func TestLogger_Fatal(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	event := logger.Fatal(ctx)

	if event == nil {
		t.Fatal("Fatal should not return nil")
	}

	var _ core.LogEvent = event
}

func TestLogger_WithContext(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.WithValue(context.Background(), "test", "value")

	newLogger := logger.WithContext(ctx)

	if newLogger == nil {
		t.Fatal("WithContext should not return nil")
	}

	if newLogger == logger {
		t.Error("WithContext should return a new logger instance")
	}

	// Verificar que o novo logger é funcional
	event := newLogger.Info(ctx)
	if event == nil {
		t.Error("New logger should be functional")
	}
}

func TestLogger_WithFields(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)

	fields := map[string]interface{}{
		"service": "auth",
		"version": "1.0.0",
	}

	newLogger := logger.WithFields(fields)

	if newLogger == nil {
		t.Fatal("WithFields should not return nil")
	}

	if newLogger == logger {
		t.Error("WithFields should return a new logger instance")
	}

	// Verificar que o novo logger é funcional
	ctx := context.Background()
	event := newLogger.Info(ctx)
	if event == nil {
		t.Error("New logger should be functional")
	}
}

func TestLogger_Integration(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	// Testar fluxo completo de logging
	logger.Info(ctx).
		Str("user_id", "123").
		Int("attempt", 1).
		Msg("User login successful")

	// Verificar que o adapter recebeu a chamada
	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]
	if call.level != core.INFO {
		t.Errorf("Expected level INFO, got %v", call.level)
	}
	if call.msg != "User login successful" {
		t.Errorf("Expected message 'User login successful', got '%s'", call.msg)
	}
	if call.ctx != ctx {
		t.Errorf("Expected context %v, got %v", ctx, call.ctx)
	}

	// Verificar campos
	if call.fields["user_id"] != "123" {
		t.Errorf("Expected field user_id='123', got %v", call.fields["user_id"])
	}
	if call.fields["attempt"] != 1 {
		t.Errorf("Expected field attempt=1, got %v", call.fields["attempt"])
	}
}

func TestLogger_WithFieldsIntegration(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)

	// Criar logger com campos pré-definidos
	serviceLogger := logger.WithFields(map[string]interface{}{
		"service": "auth",
		"version": "1.0.0",
	})

	ctx := context.Background()
	serviceLogger.Error(ctx).
		Str("error_code", "AUTH001").
		Msg("Authentication failed")

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]

	// Verificar campos pré-definidos
	if call.fields["service"] != "auth" {
		t.Errorf("Expected field service='auth', got %v", call.fields["service"])
	}
	if call.fields["version"] != "1.0.0" {
		t.Errorf("Expected field version='1.0.0', got %v", call.fields["version"])
	}

	// Verificar campo adicionado no momento do log
	if call.fields["error_code"] != "AUTH001" {
		t.Errorf("Expected field error_code='AUTH001', got %v", call.fields["error_code"])
	}
}

func TestLogger_MultipleInstances(t *testing.T) {
	adapter := &mockAdapter{}
	logger1 := New(adapter)
	logger2 := logger1.WithFields(map[string]interface{}{"instance": "2"})

	ctx := context.Background()

	// Log com logger1
	logger1.Info(ctx).Str("source", "logger1").Msg("message from logger1")

	// Log com logger2
	logger2.Info(ctx).Str("source", "logger2").Msg("message from logger2")

	if len(adapter.logCalls) != 2 {
		t.Fatalf("Expected 2 log calls, got %d", len(adapter.logCalls))
	}

	// Verificar primeira chamada (logger1)
	call1 := adapter.logCalls[0]
	if call1.fields["source"] != "logger1" {
		t.Errorf("Expected first call source='logger1', got %v", call1.fields["source"])
	}
	if _, exists := call1.fields["instance"]; exists {
		t.Error("First call should not have instance field")
	}

	// Verificar segunda chamada (logger2)
	call2 := adapter.logCalls[1]
	if call2.fields["source"] != "logger2" {
		t.Errorf("Expected second call source='logger2', got %v", call2.fields["source"])
	}
	if call2.fields["instance"] != "2" {
		t.Errorf("Expected second call instance='2', got %v", call2.fields["instance"])
	}
}

func TestLogger_AllLevels(t *testing.T) {
	adapter := &mockAdapter{}
	logger := New(adapter)
	ctx := context.Background()

	// Testar todos os níveis de log
	logger.Debug(ctx).Msg("debug message")
	logger.Info(ctx).Msg("info message")
	logger.Warn(ctx).Msg("warn message")
	logger.Error(ctx).Msg("error message")
	logger.Fatal(ctx).Msg("fatal message")

	if len(adapter.logCalls) != 5 {
		t.Fatalf("Expected 5 log calls, got %d", len(adapter.logCalls))
	}

	expectedLevels := []core.Level{core.DEBUG, core.INFO, core.WARN, core.ERROR, core.FATAL}
	expectedMessages := []string{"debug message", "info message", "warn message", "error message", "fatal message"}

	for i, call := range adapter.logCalls {
		if call.level != expectedLevels[i] {
			t.Errorf("Call %d: expected level %v, got %v", i, expectedLevels[i], call.level)
		}
		if call.msg != expectedMessages[i] {
			t.Errorf("Call %d: expected message '%s', got '%s'", i, expectedMessages[i], call.msg)
		}
	}
}

// mockAdapter para testes do logger
type mockAdapter struct {
	logCalls       []logCall
	levelEnabled   map[core.Level]bool
	contextAdapter LoggerAdapter
}

type logCall struct {
	ctx    context.Context
	level  core.Level
	msg    string
	fields map[string]interface{}
}

func (m *mockAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
	if m.logCalls == nil {
		m.logCalls = make([]logCall, 0)
	}
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

func (m *mockAdapter) IsLevelEnabled(level core.Level) bool {
	if m.levelEnabled == nil {
		return true // padrão é habilitado
	}
	enabled, exists := m.levelEnabled[level]
	if !exists {
		return true // padrão é habilitado
	}
	return enabled
}
