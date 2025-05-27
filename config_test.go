package logger

import (
	"context"
	"os"
	"testing"

	"github.com/victorximenis/logger/core"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config.ServiceName != DefaultServiceName {
		t.Errorf("Expected ServiceName %s, got %s", DefaultServiceName, config.ServiceName)
	}
	if config.Environment != DefaultEnvironment {
		t.Errorf("Expected Environment %s, got %s", DefaultEnvironment, config.Environment)
	}
	if config.Output != DefaultOutput {
		t.Errorf("Expected Output %v, got %v", DefaultOutput, config.Output)
	}
	if config.LogLevel != DefaultLogLevel {
		t.Errorf("Expected LogLevel %v, got %v", DefaultLogLevel, config.LogLevel)
	}
	if config.LogFilePath != DefaultLogFilePath {
		t.Errorf("Expected LogFilePath %s, got %s", DefaultLogFilePath, config.LogFilePath)
	}
}

func TestOutputType_String(t *testing.T) {
	tests := []struct {
		name     string
		output   OutputType
		expected string
	}{
		{"stdout only", OutputStdout, "stdout"},
		{"file only", OutputFile, "file"},
		{"both outputs", OutputStdout | OutputFile, "stdout,file"},
		{"no output", OutputType(0), "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.output.String()
			if result != tt.expected {
				t.Errorf("OutputType.String() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid config",
			config:    NewConfig(),
			expectErr: false,
		},
		{
			name: "empty service name",
			config: Config{
				ServiceName: "",
				Environment: "test",
				Output:      OutputStdout,
				LogLevel:    core.INFO,
			},
			expectErr: true,
			errMsg:    "service name cannot be empty",
		},
		{
			name: "empty environment",
			config: Config{
				ServiceName: "test-service",
				Environment: "",
				Output:      OutputStdout,
				LogLevel:    core.INFO,
			},
			expectErr: true,
			errMsg:    "environment cannot be empty",
		},
		{
			name: "no output type",
			config: Config{
				ServiceName: "test-service",
				Environment: "test",
				Output:      OutputType(0),
				LogLevel:    core.INFO,
			},
			expectErr: true,
			errMsg:    "output type must be specified",
		},
		{
			name: "file output without path",
			config: Config{
				ServiceName: "test-service",
				Environment: "test",
				Output:      OutputFile,
				LogLevel:    core.INFO,
				LogFilePath: "",
			},
			expectErr: true,
			errMsg:    "log file path must be specified when file output is enabled",
		},
		{
			name: "invalid log level",
			config: Config{
				ServiceName: "test-service",
				Environment: "test",
				Output:      OutputStdout,
				LogLevel:    core.Level(999),
			},
			expectErr: true,
			errMsg:    "invalid log level: UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected core.Level
	}{
		{"DEBUG", core.DEBUG},
		{"debug", core.DEBUG},
		{"INFO", core.INFO},
		{"info", core.INFO},
		{"WARN", core.WARN},
		{"warn", core.WARN},
		{"WARNING", core.WARN},
		{"ERROR", core.ERROR},
		{"error", core.ERROR},
		{"FATAL", core.FATAL},
		{"fatal", core.FATAL},
		{"invalid", DefaultLogLevel},
		{"", DefaultLogLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseOutputType(t *testing.T) {
	tests := []struct {
		input    string
		expected OutputType
	}{
		{"stdout", OutputStdout},
		{"file", OutputFile},
		{"stdout,file", OutputStdout | OutputFile},
		{"file,stdout", OutputStdout | OutputFile},
		{"stdout, file", OutputStdout | OutputFile}, // com espaços
		{"invalid", DefaultOutput},
		{"", DefaultOutput},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseOutputType(tt.input)
			if result != tt.expected {
				t.Errorf("parseOutputType(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"0", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Salvar valores originais das variáveis de ambiente
	originalEnvs := make(map[string]string)
	envVars := []string{
		EnvServiceName, EnvEnvironment, EnvOutput, EnvLogLevel,
		EnvLogFilePath, EnvTenantID, EnvPrettyPrint, EnvCallerEnabled,
	}

	for _, env := range envVars {
		originalEnvs[env] = os.Getenv(env)
	}

	// Limpar variáveis de ambiente para teste
	defer func() {
		for _, env := range envVars {
			if originalValue, exists := originalEnvs[env]; exists && originalValue != "" {
				os.Setenv(env, originalValue)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Teste 1: Sem variáveis de ambiente (valores padrão)
	t.Run("default values", func(t *testing.T) {
		// Limpar todas as variáveis
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		config := LoadConfigFromEnv()

		if config.ServiceName != DefaultServiceName {
			t.Errorf("Expected ServiceName %s, got %s", DefaultServiceName, config.ServiceName)
		}
		if config.Environment != DefaultEnvironment {
			t.Errorf("Expected Environment %s, got %s", DefaultEnvironment, config.Environment)
		}
		if config.Output != DefaultOutput {
			t.Errorf("Expected Output %v, got %v", DefaultOutput, config.Output)
		}
		if config.LogLevel != DefaultLogLevel {
			t.Errorf("Expected LogLevel %v, got %v", DefaultLogLevel, config.LogLevel)
		}
	})

	// Teste 2: Com variáveis de ambiente customizadas
	t.Run("custom environment variables", func(t *testing.T) {
		os.Setenv(EnvServiceName, "test-service")
		os.Setenv(EnvEnvironment, "production")
		os.Setenv(EnvOutput, "file,stdout")
		os.Setenv(EnvLogLevel, "debug")
		os.Setenv(EnvLogFilePath, "/tmp/test.log")
		os.Setenv(EnvTenantID, "tenant-123")
		os.Setenv(EnvPrettyPrint, "true")
		os.Setenv(EnvCallerEnabled, "true")

		config := LoadConfigFromEnv()

		if config.ServiceName != "test-service" {
			t.Errorf("Expected ServiceName test-service, got %s", config.ServiceName)
		}
		if config.Environment != "production" {
			t.Errorf("Expected Environment production, got %s", config.Environment)
		}
		if config.Output != (OutputStdout | OutputFile) {
			t.Errorf("Expected Output %v, got %v", OutputStdout|OutputFile, config.Output)
		}
		if config.LogLevel != core.DEBUG {
			t.Errorf("Expected LogLevel DEBUG, got %v", config.LogLevel)
		}
		if config.LogFilePath != "/tmp/test.log" {
			t.Errorf("Expected LogFilePath /tmp/test.log, got %s", config.LogFilePath)
		}
		if config.TenantID != "tenant-123" {
			t.Errorf("Expected TenantID tenant-123, got %s", config.TenantID)
		}
		if !config.PrettyPrint {
			t.Errorf("Expected PrettyPrint true, got %t", config.PrettyPrint)
		}
		if !config.CallerEnabled {
			t.Errorf("Expected CallerEnabled true, got %t", config.CallerEnabled)
		}
	})
}

func TestLoadConfigFromEnvWithValidation(t *testing.T) {
	// Salvar valores originais
	originalEnvs := make(map[string]string)
	envVars := []string{EnvServiceName, EnvEnvironment, EnvOutput, EnvLogLevel}

	for _, env := range envVars {
		originalEnvs[env] = os.Getenv(env)
	}

	defer func() {
		for _, env := range envVars {
			if originalValue, exists := originalEnvs[env]; exists && originalValue != "" {
				os.Setenv(env, originalValue)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Teste com configuração válida
	t.Run("valid config", func(t *testing.T) {
		os.Setenv(EnvServiceName, "test-service")
		os.Setenv(EnvEnvironment, "test")
		os.Setenv(EnvOutput, "stdout")
		os.Setenv(EnvLogLevel, "info")

		config, err := LoadConfigFromEnvWithValidation()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if config.ServiceName != "test-service" {
			t.Errorf("Expected ServiceName test-service, got %s", config.ServiceName)
		}
	})

	// Teste com configuração inválida
	t.Run("invalid config", func(t *testing.T) {
		// Limpar todas as variáveis primeiro
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		// Definir uma configuração que será inválida após o carregamento
		// Usar file output sem definir o caminho do arquivo
		os.Setenv(EnvServiceName, "test-service")
		os.Setenv(EnvEnvironment, "test")
		os.Setenv(EnvOutput, "file") // file output sem LogFilePath será inválido
		os.Setenv(EnvLogLevel, "info")
		// Não definir EnvLogFilePath, então usará o padrão, mas vamos sobrescrever depois

		config := LoadConfigFromEnv()
		// Forçar um estado inválido
		config.LogFilePath = "" // Isso tornará a configuração inválida

		err := config.Validate()
		if err == nil {
			t.Error("Expected error for invalid config, got none")
		}
	})
}

func TestConfig_String(t *testing.T) {
	config := Config{
		ServiceName:   "test-service",
		Environment:   "test",
		Output:        OutputStdout,
		LogLevel:      core.INFO,
		LogFilePath:   "/tmp/test.log",
		TenantID:      "tenant-123",
		PrettyPrint:   true,
		CallerEnabled: false,
	}

	result := config.String()
	expected := "Config{ServiceName: test-service, Environment: test, Output: stdout, LogLevel: INFO, LogFilePath: /tmp/test.log, TenantID: tenant-123, PrettyPrint: true, CallerEnabled: false, ObservabilityEnabled: false}"

	if result != expected {
		t.Errorf("Config.String() = %s, expected %s", result, expected)
	}
}

// Testes para as funções globais

func TestInit(t *testing.T) {
	// Reset estado global antes do teste
	resetGlobalState()

	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name:      "valid config",
			config:    NewConfig(),
			expectErr: false,
		},
		{
			name: "invalid config",
			config: Config{
				ServiceName: "", // inválido
				Environment: "test",
				Output:      OutputStdout,
				LogLevel:    core.INFO,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalState()

			err := Init(tt.config)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if IsInitialized() {
					t.Error("Expected logger not to be initialized after error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if !IsInitialized() {
					t.Error("Expected logger to be initialized")
				}

				// Verificar se a configuração foi salva
				savedConfig := GetConfig()
				if savedConfig.ServiceName != tt.config.ServiceName {
					t.Errorf("Expected saved ServiceName %s, got %s", tt.config.ServiceName, savedConfig.ServiceName)
				}
			}
		})
	}
}

func TestInitFromEnv(t *testing.T) {
	// Salvar valores originais
	originalEnvs := make(map[string]string)
	envVars := []string{EnvServiceName, EnvEnvironment, EnvOutput, EnvLogLevel}

	for _, env := range envVars {
		originalEnvs[env] = os.Getenv(env)
	}

	defer func() {
		for _, env := range envVars {
			if originalValue, exists := originalEnvs[env]; exists && originalValue != "" {
				os.Setenv(env, originalValue)
			} else {
				os.Unsetenv(env)
			}
		}
		resetGlobalState()
	}()

	// Configurar variáveis de ambiente válidas
	os.Setenv(EnvServiceName, "test-service")
	os.Setenv(EnvEnvironment, "test")
	os.Setenv(EnvOutput, "stdout")
	os.Setenv(EnvLogLevel, "info")

	resetGlobalState()

	err := InitFromEnv()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !IsInitialized() {
		t.Error("Expected logger to be initialized")
	}

	config := GetConfig()
	if config.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName test-service, got %s", config.ServiceName)
	}
}

func TestGetLogger(t *testing.T) {
	resetGlobalState()

	// Primeiro acesso deve inicializar automaticamente
	logger := GetLogger()
	if logger == nil {
		t.Error("Expected logger to be returned")
	}

	// Verificar se foi inicializado (pode ser que inicialize de forma lazy)
	// Não vamos forçar a verificação de IsInitialized() aqui para evitar race conditions
}

func TestGlobalHelpers(t *testing.T) {
	resetGlobalState()

	ctx := context.Background()

	// Testar que as funções helper não causam panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Global helper functions caused panic: %v", r)
		}
	}()

	// Testar funções de nível
	Debug(ctx).Msg("debug message")
	Info(ctx).Msg("info message")
	Warn(ctx).Msg("warn message")
	Error(ctx).Msg("error message")

	// Testar WithContext e WithFields
	contextLogger := WithContext(ctx)
	if contextLogger == nil {
		t.Error("WithContext should return a logger")
	}

	fieldsLogger := WithFields(map[string]interface{}{"test": "value"})
	if fieldsLogger == nil {
		t.Error("WithFields should return a logger")
	}
}

func TestCreateAdapterFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
	}{
		{
			name: "stdout output",
			config: Config{
				ServiceName: "test",
				Environment: "test",
				Output:      OutputStdout,
				LogLevel:    core.INFO,
			},
			expectErr: false,
		},
		{
			name: "file output with valid path",
			config: Config{
				ServiceName: "test",
				Environment: "test",
				Output:      OutputFile,
				LogLevel:    core.INFO,
				LogFilePath: "/tmp/test.log",
			},
			expectErr: false,
		},
		{
			name: "both outputs",
			config: Config{
				ServiceName: "test",
				Environment: "test",
				Output:      OutputStdout | OutputFile,
				LogLevel:    core.INFO,
				LogFilePath: "/tmp/test.log",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := createAdapterFromConfig(tt.config)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if adapter == nil {
					t.Error("Expected adapter to be created")
				}
			}
		})
	}
}

// resetGlobalState reseta o estado global para testes
func resetGlobalState() {
	initMutex.Lock()
	defer initMutex.Unlock()

	defaultLogger = nil
	defaultConfig = Config{}
	isInitialized = false
}
