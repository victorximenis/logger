package core

import (
	"context"
	"errors"
	"testing"
)

func TestNewLogEvent(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	level := INFO

	event := NewLogEvent(adapter, ctx, level)

	if event == nil {
		t.Fatal("NewLogEvent should not return nil")
	}

	// Verificar que o evento implementa a interface LogEvent
	var _ LogEvent = event
}

func TestLogEvent_Str(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	result := event.Str("key", "value")

	// Verificar que retorna o próprio evento para method chaining
	if result != event {
		t.Error("Str should return the same event for method chaining")
	}

	// Testar que o campo foi adicionado
	event.Msg("test message")

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	fields := adapter.logCalls[0].fields
	if fields["key"] != "value" {
		t.Errorf("Expected field key='value', got %v", fields["key"])
	}
}

func TestLogEvent_Int(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	result := event.Int("count", 42)

	if result != event {
		t.Error("Int should return the same event for method chaining")
	}

	event.Msg("test message")

	fields := adapter.logCalls[0].fields
	if fields["count"] != 42 {
		t.Errorf("Expected field count=42, got %v", fields["count"])
	}
}

func TestLogEvent_Float64(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	result := event.Float64("price", 19.99)

	if result != event {
		t.Error("Float64 should return the same event for method chaining")
	}

	event.Msg("test message")

	fields := adapter.logCalls[0].fields
	if fields["price"] != 19.99 {
		t.Errorf("Expected field price=19.99, got %v", fields["price"])
	}
}

func TestLogEvent_Bool(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	result := event.Bool("active", true)

	if result != event {
		t.Error("Bool should return the same event for method chaining")
	}

	event.Msg("test message")

	fields := adapter.logCalls[0].fields
	if fields["active"] != true {
		t.Errorf("Expected field active=true, got %v", fields["active"])
	}
}

func TestLogEvent_Err(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, ERROR)

	testErr := errors.New("test error")
	result := event.Err(testErr)

	if result != event {
		t.Error("Err should return the same event for method chaining")
	}

	event.Msg("error occurred")

	fields := adapter.logCalls[0].fields
	if fields["error"] != "test error" {
		t.Errorf("Expected field error='test error', got %v", fields["error"])
	}
}

func TestLogEvent_Err_Nil(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, ERROR)

	result := event.Err(nil)

	if result != event {
		t.Error("Err should return the same event for method chaining")
	}

	event.Msg("no error")

	fields := adapter.logCalls[0].fields
	if _, exists := fields["error"]; exists {
		t.Error("Error field should not be added when err is nil")
	}
}

func TestLogEvent_Any(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	testStruct := struct {
		Name string
		Age  int
	}{"John", 30}

	result := event.Any("user", testStruct)

	if result != event {
		t.Error("Any should return the same event for method chaining")
	}

	event.Msg("user data")

	fields := adapter.logCalls[0].fields
	if fields["user"] != testStruct {
		t.Errorf("Expected field user=%v, got %v", testStruct, fields["user"])
	}
}

func TestLogEvent_Fields(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	fieldsToAdd := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	result := event.Fields(fieldsToAdd)

	if result != event {
		t.Error("Fields should return the same event for method chaining")
	}

	event.Msg("multiple fields")

	fields := adapter.logCalls[0].fields
	for k, v := range fieldsToAdd {
		if fields[k] != v {
			t.Errorf("Expected field %s=%v, got %v", k, v, fields[k])
		}
	}
}

func TestLogEvent_Msg(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	event.Str("key", "value").Msg("test message")

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]
	if call.msg != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", call.msg)
	}
	if call.level != INFO {
		t.Errorf("Expected level INFO, got %v", call.level)
	}
	if call.ctx != ctx {
		t.Errorf("Expected context %v, got %v", ctx, call.ctx)
	}
}

func TestLogEvent_Msgf(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	event.Str("key", "value").Msgf("user %s has %d points", "John", 100)

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]
	expectedMsg := "user John has 100 points"
	if call.msg != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, call.msg)
	}
}

func TestLogEvent_Send(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	event.Str("key", "value").Send()

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	call := adapter.logCalls[0]
	if call.msg != "" {
		t.Errorf("Expected empty message, got '%s'", call.msg)
	}
}

func TestLogEvent_MethodChaining(t *testing.T) {
	adapter := newMockAdapter()
	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	// Testar method chaining completo
	event.
		Str("service", "auth").
		Int("user_id", 123).
		Float64("duration", 1.5).
		Bool("success", true).
		Any("metadata", map[string]string{"version": "1.0"}).
		Fields(map[string]interface{}{"extra": "data"}).
		Msg("operation completed")

	if len(adapter.logCalls) != 1 {
		t.Fatalf("Expected 1 log call, got %d", len(adapter.logCalls))
	}

	fields := adapter.logCalls[0].fields

	// Verificar campos individualmente para evitar comparação direta de maps
	if fields["service"] != "auth" {
		t.Errorf("Expected field service='auth', got %v", fields["service"])
	}
	if fields["user_id"] != 123 {
		t.Errorf("Expected field user_id=123, got %v", fields["user_id"])
	}
	if fields["duration"] != 1.5 {
		t.Errorf("Expected field duration=1.5, got %v", fields["duration"])
	}
	if fields["success"] != true {
		t.Errorf("Expected field success=true, got %v", fields["success"])
	}
	if fields["extra"] != "data" {
		t.Errorf("Expected field extra='data', got %v", fields["extra"])
	}

	// Verificar o campo metadata separadamente
	metadata, ok := fields["metadata"].(map[string]string)
	if !ok {
		t.Errorf("Expected metadata to be map[string]string, got %T", fields["metadata"])
	} else if metadata["version"] != "1.0" {
		t.Errorf("Expected metadata version='1.0', got %v", metadata["version"])
	}
}

func TestLogEvent_LevelDisabled(t *testing.T) {
	adapter := newMockAdapter()
	adapter.setLevelEnabled(DEBUG, false)

	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, DEBUG)

	event.Str("key", "value").Msg("debug message")

	// Não deve haver chamadas de log quando o nível está desabilitado
	if len(adapter.logCalls) != 0 {
		t.Errorf("Expected 0 log calls when level is disabled, got %d", len(adapter.logCalls))
	}
}

func TestLogEvent_LevelEnabled(t *testing.T) {
	adapter := newMockAdapter()
	adapter.setLevelEnabled(INFO, true)

	ctx := context.Background()
	event := NewLogEvent(adapter, ctx, INFO)

	event.Str("key", "value").Msg("info message")

	// Deve haver uma chamada de log quando o nível está habilitado
	if len(adapter.logCalls) != 1 {
		t.Errorf("Expected 1 log call when level is enabled, got %d", len(adapter.logCalls))
	}
}
