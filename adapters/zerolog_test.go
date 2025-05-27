package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/victorximenis/logger/core"
)

func TestNewZerologAdapter(t *testing.T) {
	tests := []struct {
		name   string
		config *ZerologConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &ZerologConfig{
				Level:         core.DEBUG,
				PrettyPrint:   true,
				CallerEnabled: true,
			},
		},
		{
			name: "with minimal config",
			config: &ZerologConfig{
				Level: core.ERROR,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewZerologAdapter(tt.config)
			if adapter == nil {
				t.Fatal("NewZerologAdapter should not return nil")
			}

			// Verificar que implementa a interface LoggerAdapter
			var _ core.LoggerAdapter = adapter
		})
	}
}

func TestNewZerologAdapterFromLogger(t *testing.T) {
	logger := zerolog.New(nil)
	adapter := NewZerologAdapterFromLogger(logger)

	if adapter == nil {
		t.Fatal("NewZerologAdapterFromLogger should not return nil")
	}

	// Verificar que implementa a interface LoggerAdapter
	var _ core.LoggerAdapter = adapter
}

func TestZerologAdapter_Log(t *testing.T) {
	tests := []struct {
		name     string
		level    core.Level
		msg      string
		fields   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:  "info level with message",
			level: core.INFO,
			msg:   "test message",
			fields: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			expected: map[string]interface{}{
				"level":   "INFO",
				"message": "test message",
				"key1":    "value1",
				"key2":    float64(42), // JSON unmarshaling converts numbers to float64
			},
		},
		{
			name:  "error level with error field",
			level: core.ERROR,
			msg:   "error occurred",
			fields: map[string]interface{}{
				"error": errors.New("test error"),
				"code":  500,
			},
			expected: map[string]interface{}{
				"level":   "ERROR",
				"message": "error occurred",
				"error":   "test error",
				"code":    float64(500),
			},
		},
		{
			name:  "debug level with various field types",
			level: core.DEBUG,
			msg:   "debug info",
			fields: map[string]interface{}{
				"string_field": "test",
				"int_field":    123,
				"float_field":  3.14,
				"bool_field":   true,
			},
			expected: map[string]interface{}{
				"level":        "DEBUG",
				"message":      "debug info",
				"string_field": "test",
				"int_field":    float64(123),
				"float_field":  3.14,
				"bool_field":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := &ZerologConfig{
				Writer:      &buf,
				Level:       core.DEBUG, // Permitir todos os níveis
				TimeFormat:  "",         // Desabilitar timestamp para testes
				PrettyPrint: false,
			}

			adapter := NewZerologAdapter(config)
			ctx := context.Background()

			adapter.Log(ctx, tt.level, tt.msg, tt.fields)

			// Verificar se algo foi escrito
			output := buf.String()
			if output == "" {
				t.Fatal("Expected log output, got empty string")
			}

			// Parse JSON output
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
				t.Fatalf("Failed to parse log output as JSON: %v\nOutput: %s", err, output)
			}

			// Verificar campos esperados
			for key, expectedValue := range tt.expected {
				actualValue, exists := logEntry[key]
				if !exists {
					t.Errorf("Expected field %s not found in log output", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("Field %s: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestZerologAdapter_WithContext(t *testing.T) {
	var buf bytes.Buffer
	config := &ZerologConfig{
		Writer:     &buf,
		Level:      core.DEBUG,
		TimeFormat: "",
	}

	adapter := NewZerologAdapter(config)
	ctx := context.WithValue(context.Background(), "test_key", "test_value")

	// Criar novo adapter com contexto
	newAdapter := adapter.WithContext(ctx)

	if newAdapter == adapter {
		t.Error("WithContext should return a new adapter instance")
	}

	// Verificar que o novo adapter é funcional
	newAdapter.Log(ctx, core.INFO, "test message", nil)

	output := buf.String()
	if output == "" {
		t.Fatal("Expected log output from new adapter")
	}
}

func TestZerologAdapter_IsLevelEnabled(t *testing.T) {
	tests := []struct {
		name         string
		configLevel  core.Level
		testLevel    core.Level
		shouldEnable bool
	}{
		{
			name:         "debug level enables debug",
			configLevel:  core.DEBUG,
			testLevel:    core.DEBUG,
			shouldEnable: true,
		},
		{
			name:         "debug level enables info",
			configLevel:  core.DEBUG,
			testLevel:    core.INFO,
			shouldEnable: true,
		},
		{
			name:         "info level disables debug",
			configLevel:  core.INFO,
			testLevel:    core.DEBUG,
			shouldEnable: false,
		},
		{
			name:         "info level enables info",
			configLevel:  core.INFO,
			testLevel:    core.INFO,
			shouldEnable: true,
		},
		{
			name:         "error level disables info",
			configLevel:  core.ERROR,
			testLevel:    core.INFO,
			shouldEnable: false,
		},
		{
			name:         "error level enables error",
			configLevel:  core.ERROR,
			testLevel:    core.ERROR,
			shouldEnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ZerologConfig{
				Level: tt.configLevel,
			}

			adapter := NewZerologAdapter(config)
			enabled := adapter.IsLevelEnabled(tt.testLevel)

			if enabled != tt.shouldEnable {
				t.Errorf("IsLevelEnabled(%v) with config level %v: expected %v, got %v",
					tt.testLevel, tt.configLevel, tt.shouldEnable, enabled)
			}
		})
	}
}

func TestMapLevelToZerolog(t *testing.T) {
	tests := []struct {
		input    core.Level
		expected zerolog.Level
	}{
		{core.DEBUG, zerolog.DebugLevel},
		{core.INFO, zerolog.InfoLevel},
		{core.WARN, zerolog.WarnLevel},
		{core.ERROR, zerolog.ErrorLevel},
		{core.FATAL, zerolog.FatalLevel},
		{core.Level(999), zerolog.InfoLevel}, // Unknown level defaults to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := mapLevelToZerolog(tt.input)
			if result != tt.expected {
				t.Errorf("mapLevelToZerolog(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestZerologAdapter_LogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := &ZerologConfig{
		Writer:     &buf,
		Level:      core.WARN, // Só permitir WARN e acima
		TimeFormat: "",
	}

	adapter := NewZerologAdapter(config)
	ctx := context.Background()

	// Tentar log DEBUG (deve ser filtrado)
	adapter.Log(ctx, core.DEBUG, "debug message", nil)
	debugOutput := buf.String()

	// Tentar log INFO (deve ser filtrado)
	buf.Reset()
	adapter.Log(ctx, core.INFO, "info message", nil)
	infoOutput := buf.String()

	// Tentar log WARN (deve passar)
	buf.Reset()
	adapter.Log(ctx, core.WARN, "warn message", nil)
	warnOutput := buf.String()

	// Verificar filtragem
	if debugOutput != "" {
		t.Error("DEBUG log should be filtered out")
	}
	if infoOutput != "" {
		t.Error("INFO log should be filtered out")
	}
	if warnOutput == "" {
		t.Error("WARN log should not be filtered out")
	}
}

func TestZerologAdapter_Integration(t *testing.T) {
	var buf bytes.Buffer
	config := &ZerologConfig{
		Writer:     &buf,
		Level:      core.DEBUG,
		TimeFormat: "",
	}

	adapter := NewZerologAdapter(config)
	ctx := context.Background()

	// Teste de integração completo
	fields := map[string]interface{}{
		"user_id":    "123",
		"session_id": "abc-def",
		"action":     "login",
		"success":    true,
		"duration":   1.5,
	}

	adapter.Log(ctx, core.INFO, "User login completed", fields)

	output := buf.String()
	if output == "" {
		t.Fatal("Expected log output")
	}

	// Verificar que contém informações esperadas
	expectedStrings := []string{
		"User login completed",
		"user_id",
		"123",
		"session_id",
		"abc-def",
		"action",
		"login",
		"success",
		"duration",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected log output to contain '%s', but it didn't.\nOutput: %s", expected, output)
		}
	}
}
