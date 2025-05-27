package logger

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/victorximenis/logger/adapters"
	"github.com/victorximenis/logger/core"
	"github.com/victorximenis/logger/observability"
)

// OutputType define os tipos de saída de log disponíveis
type OutputType int

const (
	// OutputStdout direciona logs para stdout
	OutputStdout OutputType = 1 << iota
	// OutputFile direciona logs para arquivo
	OutputFile
)

// String retorna a representação em string do tipo de saída
func (o OutputType) String() string {
	var outputs []string
	if o&OutputStdout != 0 {
		outputs = append(outputs, "stdout")
	}
	if o&OutputFile != 0 {
		outputs = append(outputs, "file")
	}
	if len(outputs) == 0 {
		return "none"
	}
	return strings.Join(outputs, ",")
}

// Config define a configuração do sistema de logging
type Config struct {
	// ServiceName é o nome do serviço que está gerando os logs
	ServiceName string
	// Environment é o ambiente onde o serviço está executando (development, staging, production)
	Environment string
	// Output define onde os logs serão direcionados (stdout, file, ou ambos)
	Output OutputType
	// LogLevel define o nível mínimo de log que será registrado
	LogLevel core.Level
	// LogFilePath define o caminho do arquivo de log quando Output inclui OutputFile
	LogFilePath string
	// TenantID é um identificador opcional para multi-tenancy
	TenantID string
	// PrettyPrint habilita formatação legível para desenvolvimento
	PrettyPrint bool
	// CallerEnabled habilita informações do caller nos logs
	CallerEnabled bool
	// Observability define as configurações de observabilidade
	Observability observability.ObservabilityConfig
}

// Constantes para valores padrão
const (
	// DefaultServiceName é o nome padrão do serviço
	DefaultServiceName = "unknown-service"
	// DefaultEnvironment é o ambiente padrão
	DefaultEnvironment = "development"
	// DefaultLogLevel é o nível de log padrão
	DefaultLogLevel = core.INFO
	// DefaultLogFilePath é o caminho padrão do arquivo de log
	DefaultLogFilePath = "/var/log/app.log"
	// DefaultOutput é o tipo de saída padrão
	DefaultOutput = OutputStdout
)

// Constantes para nomes de variáveis de ambiente
const (
	// EnvServiceName é o nome da variável de ambiente para o nome do serviço
	EnvServiceName = "LOGGER_SERVICE_NAME"
	// EnvEnvironment é o nome da variável de ambiente para o ambiente
	EnvEnvironment = "LOGGER_ENVIRONMENT"
	// EnvOutput é o nome da variável de ambiente para o tipo de saída
	EnvOutput = "LOGGER_OUTPUT"
	// EnvLogLevel é o nome da variável de ambiente para o nível de log
	EnvLogLevel = "LOGGER_LOG_LEVEL"
	// EnvLogFilePath é o nome da variável de ambiente para o caminho do arquivo de log
	EnvLogFilePath = "LOGGER_LOG_FILE_PATH"
	// EnvTenantID é o nome da variável de ambiente para o tenant ID
	EnvTenantID = "LOGGER_TENANT_ID"
	// EnvPrettyPrint é o nome da variável de ambiente para pretty print
	EnvPrettyPrint = "LOGGER_PRETTY_PRINT"
	// EnvCallerEnabled é o nome da variável de ambiente para habilitar caller
	EnvCallerEnabled = "LOGGER_CALLER_ENABLED"
	// EnvObservabilityEnabled é o nome da variável de ambiente para habilitar observabilidade
	EnvObservabilityEnabled = "LOGGER_OBSERVABILITY_ENABLED"
)

// Variáveis globais para o logger padrão
var (
	defaultLogger Logger
	defaultConfig Config
	initMutex     sync.RWMutex
	isInitialized bool
)

// NewConfig cria uma nova configuração com valores padrão
func NewConfig() Config {
	return Config{
		ServiceName:   DefaultServiceName,
		Environment:   DefaultEnvironment,
		Output:        DefaultOutput,
		LogLevel:      DefaultLogLevel,
		LogFilePath:   DefaultLogFilePath,
		TenantID:      "",
		PrettyPrint:   false,
		CallerEnabled: false,
		Observability: observability.DefaultObservabilityConfig(),
	}
}

// LoadConfigFromEnv carrega a configuração a partir de variáveis de ambiente
// com fallback para valores padrão quando as variáveis não estão definidas
func LoadConfigFromEnv() Config {
	// Carregar configuração base de observabilidade
	observabilityConfig := observability.DefaultObservabilityConfig()

	// Sobrescrever com configurações específicas do logger se definidas
	if parseBool(getEnv(EnvObservabilityEnabled, "")) {
		observabilityConfig.Enabled = true
	}

	config := Config{
		ServiceName:   getEnv(EnvServiceName, DefaultServiceName),
		Environment:   getEnv(EnvEnvironment, DefaultEnvironment),
		Output:        parseOutputType(getEnv(EnvOutput, "stdout")),
		LogLevel:      parseLogLevel(getEnv(EnvLogLevel, "info")),
		LogFilePath:   getEnv(EnvLogFilePath, DefaultLogFilePath),
		TenantID:      getEnv(EnvTenantID, ""),
		PrettyPrint:   parseBool(getEnv(EnvPrettyPrint, "false")),
		CallerEnabled: parseBool(getEnv(EnvCallerEnabled, "false")),
		Observability: observabilityConfig,
	}

	// Sincronizar configurações entre logger e observabilidade
	config.Observability.Datadog.ServiceName = config.ServiceName
	config.Observability.Datadog.Environment = config.Environment
	config.Observability.ELK.ServiceName = config.ServiceName
	config.Observability.ELK.Environment = config.Environment

	return config
}

// LoadConfigFromEnvWithValidation carrega a configuração de variáveis de ambiente
// e valida os valores carregados, retornando erro se a configuração for inválida
func LoadConfigFromEnvWithValidation() (Config, error) {
	config := LoadConfigFromEnv()

	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration loaded from environment: %w", err)
	}

	return config, nil
}

// Init inicializa o logger global com a configuração especificada
// Esta função é thread-safe e pode ser chamada múltiplas vezes
func Init(config Config) error {
	initMutex.Lock()
	defer initMutex.Unlock()

	// Validar configuração
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Criar adapter baseado na configuração
	adapter, err := createAdapterFromConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Criar logger com campos pré-definidos baseados na configuração
	preDefinedFields := map[string]interface{}{
		"service":     config.ServiceName,
		"environment": config.Environment,
	}

	if config.TenantID != "" {
		preDefinedFields["tenant_id"] = config.TenantID
	}

	// Criar logger global
	defaultLogger = New(adapter).WithFields(preDefinedFields)
	defaultConfig = config
	isInitialized = true

	return nil
}

// InitFromEnv inicializa o logger global carregando a configuração de variáveis de ambiente
func InitFromEnv() error {
	config, err := LoadConfigFromEnvWithValidation()
	if err != nil {
		return err
	}

	return Init(config)
}

// IsInitialized retorna true se o logger global foi inicializado
func IsInitialized() bool {
	initMutex.RLock()
	defer initMutex.RUnlock()
	return isInitialized
}

// GetConfig retorna a configuração atual do logger global
func GetConfig() Config {
	initMutex.RLock()
	defer initMutex.RUnlock()
	return defaultConfig
}

// GetLogger retorna o logger global. Se não foi inicializado, inicializa com configuração padrão
func GetLogger() Logger {
	initMutex.RLock()
	if isInitialized {
		logger := defaultLogger
		initMutex.RUnlock()
		return logger
	}
	initMutex.RUnlock()

	// Se não foi inicializado, inicializar com configuração padrão
	initMutex.Lock()
	defer initMutex.Unlock()

	// Verificar novamente após obter o lock de escrita
	if isInitialized {
		return defaultLogger
	}

	// Criar logger básico diretamente sem chamar Init para evitar deadlock
	config := NewConfig()
	adapter := adapters.NewZerologAdapter(&adapters.ZerologConfig{
		Level:         config.LogLevel,
		PrettyPrint:   config.PrettyPrint,
		CallerEnabled: config.CallerEnabled,
	})

	// Criar logger com campos pré-definidos baseados na configuração
	preDefinedFields := map[string]interface{}{
		"service":     config.ServiceName,
		"environment": config.Environment,
	}

	defaultLogger = New(adapter).WithFields(preDefinedFields)
	defaultConfig = config
	isInitialized = true

	return defaultLogger
}

// Funções helper globais para logging

// Debug retorna um LogEvent para nível DEBUG usando o logger global
func Debug(ctx context.Context) core.LogEvent {
	return GetLogger().Debug(ctx)
}

// Info retorna um LogEvent para nível INFO usando o logger global
func Info(ctx context.Context) core.LogEvent {
	return GetLogger().Info(ctx)
}

// Warn retorna um LogEvent para nível WARN usando o logger global
func Warn(ctx context.Context) core.LogEvent {
	return GetLogger().Warn(ctx)
}

// Error retorna um LogEvent para nível ERROR usando o logger global
func Error(ctx context.Context) core.LogEvent {
	return GetLogger().Error(ctx)
}

// Fatal retorna um LogEvent para nível FATAL usando o logger global
func Fatal(ctx context.Context) core.LogEvent {
	return GetLogger().Fatal(ctx)
}

// WithContext retorna um novo logger global com contexto
func WithContext(ctx context.Context) Logger {
	return GetLogger().WithContext(ctx)
}

// WithFields retorna um novo logger global com campos pré-definidos
func WithFields(fields map[string]interface{}) Logger {
	return GetLogger().WithFields(fields)
}

// createAdapterFromConfig cria um adapter baseado na configuração
func createAdapterFromConfig(config Config) (core.LoggerAdapter, error) {
	// Criar adapter base (Zerolog)
	baseAdapter, err := createBaseAdapter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create base adapter: %w", err)
	}

	// Se observabilidade está desabilitada, retornar apenas o adapter base
	if !config.Observability.Enabled {
		return baseAdapter, nil
	}

	// Criar adapter de observabilidade baseado no ambiente
	var observabilityAdapter core.LoggerAdapter
	switch strings.ToLower(config.Environment) {
	case "production", "prod":
		observabilityAdapter, err = observability.NewProductionObservabilityAdapter(baseAdapter)
	case "development", "dev":
		observabilityAdapter, err = observability.NewDevelopmentObservabilityAdapter(baseAdapter)
	default:
		// Usar configuração personalizada para outros ambientes
		observabilityAdapter, err = observability.NewCustomObservabilityAdapter(baseAdapter, config.Observability)
	}

	if err != nil {
		// Se falhar ao criar adapter de observabilidade, usar apenas o base
		return baseAdapter, nil
	}

	return observabilityAdapter, nil
}

// createBaseAdapter cria o adapter base (Zerolog) baseado na configuração
func createBaseAdapter(config Config) (core.LoggerAdapter, error) {
	// Configurar ZerologConfig baseado na Config
	zerologConfig := &adapters.ZerologConfig{
		Level:         config.LogLevel,
		PrettyPrint:   config.PrettyPrint,
		CallerEnabled: config.CallerEnabled,
	}

	// Configurar saída usando OutputManager
	var outputConfig core.OutputConfig

	if config.Output&OutputFile != 0 {
		// Configurar para saída em arquivo
		outputConfig = core.NewOutputConfig(config.LogFilePath)
		// Usar configurações padrão do OutputManager para rotação
	} else {
		// Configurar para stdout apenas (sem arquivo, mas com valores padrão válidos)
		outputConfig = core.OutputConfig{
			FilePath:   "", // Sem arquivo
			MaxSize:    core.DefaultMaxSize,
			MaxAge:     core.DefaultMaxAge,
			MaxBackups: core.DefaultMaxBackups,
			Compress:   core.DefaultCompress,
			LocalTime:  core.DefaultLocalTime,
		}
	}

	// Criar OutputManager
	outputManager, err := core.NewOutputManager(outputConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create output manager: %w", err)
	}

	// Configurar writer baseado no tipo de output
	if config.Output == OutputStdout {
		zerologConfig.Writer = outputManager.GetWriter() // stdout
	} else if config.Output == OutputFile {
		zerologConfig.Writer = outputManager.GetWriter() // arquivo
	} else if config.Output == (OutputStdout | OutputFile) {
		zerologConfig.Writer = outputManager.GetMultiWriter() // ambos
	} else {
		zerologConfig.Writer = os.Stdout // fallback
	}

	return adapters.NewZerologAdapter(zerologConfig), nil
}

// String retorna uma representação em string da configuração para debugging
func (c Config) String() string {
	return fmt.Sprintf("Config{ServiceName: %s, Environment: %s, Output: %s, LogLevel: %s, LogFilePath: %s, TenantID: %s, PrettyPrint: %t, CallerEnabled: %t, ObservabilityEnabled: %t}",
		c.ServiceName, c.Environment, c.Output.String(), c.LogLevel.String(), c.LogFilePath, c.TenantID, c.PrettyPrint, c.CallerEnabled, c.Observability.Enabled)
}

// Validate verifica se a configuração é válida
func (c Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if c.Environment == "" {
		return fmt.Errorf("environment cannot be empty")
	}

	if c.Output == 0 {
		return fmt.Errorf("output type must be specified")
	}

	if c.Output&OutputFile != 0 && c.LogFilePath == "" {
		return fmt.Errorf("log file path must be specified when file output is enabled")
	}

	// Validar nível de log
	switch c.LogLevel {
	case core.DEBUG, core.INFO, core.WARN, core.ERROR, core.FATAL:
		// Níveis válidos
	default:
		return fmt.Errorf("invalid log level: %v", c.LogLevel)
	}

	return nil
}

// getEnv obtém uma variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseLogLevel converte uma string para core.Level
func parseLogLevel(levelStr string) core.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return core.DEBUG
	case "INFO":
		return core.INFO
	case "WARN", "WARNING":
		return core.WARN
	case "ERROR":
		return core.ERROR
	case "FATAL":
		return core.FATAL
	default:
		return DefaultLogLevel
	}
}

// parseOutputType converte uma string para OutputType
func parseOutputType(outputStr string) OutputType {
	var output OutputType
	parts := strings.Split(strings.ToLower(outputStr), ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "stdout":
			output |= OutputStdout
		case "file":
			output |= OutputFile
		}
	}

	if output == 0 {
		return DefaultOutput
	}

	return output
}

// parseBool converte uma string para bool
func parseBool(boolStr string) bool {
	if boolStr == "" {
		return false
	}

	value, err := strconv.ParseBool(boolStr)
	if err != nil {
		return false
	}

	return value
}

// Funções helper para diferentes perfis de configuração

// NewProductionConfig cria uma configuração otimizada para produção
func NewProductionConfig(serviceName string) Config {
	config := NewConfig()
	config.ServiceName = serviceName
	config.Environment = "production"
	config.LogLevel = core.INFO
	config.PrettyPrint = false
	config.CallerEnabled = false

	// Configurações de observabilidade para produção
	config.Observability.Enabled = true
	config.Observability.EnableDatadog = true
	config.Observability.EnableELK = true
	config.Observability.EnableCorrelationID = true
	config.Observability.FallbackOnError = true

	return config
}

// NewDevelopmentConfig cria uma configuração otimizada para desenvolvimento
func NewDevelopmentConfig(serviceName string) Config {
	config := NewConfig()
	config.ServiceName = serviceName
	config.Environment = "development"
	config.LogLevel = core.DEBUG
	config.PrettyPrint = true
	config.CallerEnabled = true

	// Configurações de observabilidade para desenvolvimento
	config.Observability.Enabled = true
	config.Observability.EnableDatadog = false // Desabilitado em dev
	config.Observability.EnableELK = true
	config.Observability.EnableCorrelationID = true
	config.Observability.FallbackOnError = true

	return config
}

// NewStagingConfig cria uma configuração otimizada para staging
func NewStagingConfig(serviceName string) Config {
	config := NewConfig()
	config.ServiceName = serviceName
	config.Environment = "staging"
	config.LogLevel = core.DEBUG
	config.PrettyPrint = false
	config.CallerEnabled = true

	// Configurações de observabilidade para staging
	config.Observability.Enabled = true
	config.Observability.EnableDatadog = true
	config.Observability.EnableELK = true
	config.Observability.EnableCorrelationID = true
	config.Observability.FallbackOnError = true

	return config
}

// InitWithProfile inicializa o logger com um perfil específico
func InitWithProfile(profile string, serviceName string) error {
	var config Config

	switch strings.ToLower(profile) {
	case "production", "prod":
		config = NewProductionConfig(serviceName)
	case "development", "dev":
		config = NewDevelopmentConfig(serviceName)
	case "staging", "stage":
		config = NewStagingConfig(serviceName)
	default:
		return fmt.Errorf("unknown profile: %s", profile)
	}

	return Init(config)
}

// InitWithObservability inicializa o logger com observabilidade habilitada
func InitWithObservability(serviceName, environment string, enableDatadog, enableELK bool) error {
	config := NewConfig()
	config.ServiceName = serviceName
	config.Environment = environment

	// Configurar observabilidade
	config.Observability.Enabled = true
	config.Observability.EnableDatadog = enableDatadog
	config.Observability.EnableELK = enableELK
	config.Observability.EnableCorrelationID = true
	config.Observability.FallbackOnError = true

	return Init(config)
}
