package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// RotationEvent representa um evento de rotação de log
type RotationEvent struct {
	Timestamp time.Time
	OldFile   string
	NewFile   string
	FileSize  int64
	Success   bool
	Error     error
}

// RotationHook é uma função que é chamada quando ocorre um evento de rotação
type RotationHook func(event RotationEvent)

// OutputConfig define as configurações para saída de logs
type OutputConfig struct {
	// FilePath é o caminho do arquivo de log
	FilePath string
	// MaxSize é o tamanho máximo do arquivo em megabytes antes da rotação
	MaxSize int
	// MaxAge é o número máximo de dias para manter arquivos antigos
	MaxAge int
	// MaxBackups é o número máximo de arquivos de backup antigos para manter
	MaxBackups int
	// Compress determina se os arquivos rotacionados devem ser comprimidos
	Compress bool
	// LocalTime determina se deve usar horário local para timestamps nos nomes dos arquivos
	LocalTime bool
}

// OutputManager gerencia a saída de logs para diferentes destinos
type OutputManager struct {
	config        OutputConfig
	fileWriter    io.WriteCloser
	isFileMode    bool
	rotationHooks []RotationHook
	mu            sync.RWMutex
	lastRotation  time.Time
	rotationCount int64
}

// Constantes para valores padrão
const (
	// DefaultMaxSize é o tamanho máximo padrão do arquivo em MB
	DefaultMaxSize = 100
	// DefaultMaxAge é a idade máxima padrão dos arquivos em dias
	DefaultMaxAge = 7
	// DefaultMaxBackups é o número máximo padrão de backups
	DefaultMaxBackups = 5
	// DefaultCompress define se deve comprimir por padrão
	DefaultCompress = true
	// DefaultLocalTime define se deve usar horário local por padrão
	DefaultLocalTime = false
)

// NewOutputConfig cria uma nova configuração de saída com valores padrão
func NewOutputConfig(filePath string) OutputConfig {
	return OutputConfig{
		FilePath:   filePath,
		MaxSize:    DefaultMaxSize,
		MaxAge:     DefaultMaxAge,
		MaxBackups: DefaultMaxBackups,
		Compress:   DefaultCompress,
		LocalTime:  DefaultLocalTime,
	}
}

// NewOutputManager cria um novo gerenciador de saída
func NewOutputManager(config OutputConfig) (*OutputManager, error) {
	om := &OutputManager{
		config: config,
	}

	// Validar configuração
	if err := om.validateConfig(); err != nil {
		return nil, fmt.Errorf("invalid output configuration: %w", err)
	}

	// Configurar saída de arquivo se especificada
	if config.FilePath != "" {
		if err := om.setupFileOutput(); err != nil {
			return nil, fmt.Errorf("failed to setup file output: %w", err)
		}
		om.isFileMode = true
	}

	return om, nil
}

// validateConfig valida a configuração de saída
func (om *OutputManager) validateConfig() error {
	if om.config.FilePath != "" {
		// Verificar se o diretório pai é válido
		dir := filepath.Dir(om.config.FilePath)
		if dir == "" || dir == "." {
			return fmt.Errorf("invalid file path: directory cannot be empty")
		}

		// Verificar se o caminho termina com "/" (indicando diretório, não arquivo)
		if strings.HasSuffix(om.config.FilePath, "/") || strings.HasSuffix(om.config.FilePath, "\\") {
			return fmt.Errorf("invalid file path: filename cannot be empty")
		}

		// Verificar se o nome do arquivo é válido
		filename := filepath.Base(om.config.FilePath)
		if filename == "" || filename == "." || filename == ".." {
			return fmt.Errorf("invalid file path: filename cannot be empty")
		}
	}

	if om.config.MaxSize <= 0 {
		return fmt.Errorf("max size must be positive, got %d", om.config.MaxSize)
	}

	if om.config.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative, got %d", om.config.MaxAge)
	}

	if om.config.MaxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative, got %d", om.config.MaxBackups)
	}

	return nil
}

// setupFileOutput configura a saída para arquivo com rotação
func (om *OutputManager) setupFileOutput() error {
	// Garantir que o diretório existe
	dir := filepath.Dir(om.config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	// Configurar lumberjack para rotação de logs
	lumberjackLogger := &lumberjack.Logger{
		Filename:   om.config.FilePath,
		MaxSize:    om.config.MaxSize,
		MaxAge:     om.config.MaxAge,
		MaxBackups: om.config.MaxBackups,
		Compress:   om.config.Compress,
		LocalTime:  om.config.LocalTime,
	}

	om.fileWriter = lumberjackLogger
	return nil
}

// GetWriter retorna o writer apropriado baseado na configuração
func (om *OutputManager) GetWriter() io.Writer {
	if om.isFileMode && om.fileWriter != nil {
		return om.fileWriter
	}

	// Fallback para stdout se não há configuração de arquivo
	return os.Stdout
}

// GetMultiWriter retorna um MultiWriter que escreve tanto para stdout quanto para arquivo
func (om *OutputManager) GetMultiWriter() io.Writer {
	if om.isFileMode && om.fileWriter != nil {
		return io.MultiWriter(os.Stdout, om.fileWriter)
	}

	// Se não há arquivo configurado, retorna apenas stdout
	return os.Stdout
}

// Close fecha o writer de arquivo se estiver aberto
func (om *OutputManager) Close() error {
	if om.fileWriter != nil {
		return om.fileWriter.Close()
	}
	return nil
}

// Rotate força a rotação do arquivo de log atual
func (om *OutputManager) Rotate() error {
	// Usar o método com recuperação para maior robustez
	return om.RotateWithRecovery()
}

// GetConfig retorna a configuração atual
func (om *OutputManager) GetConfig() OutputConfig {
	return om.config
}

// UpdateConfig atualiza a configuração em tempo de execução
func (om *OutputManager) UpdateConfig(newConfig OutputConfig) error {
	// Validar nova configuração
	tempOM := &OutputManager{config: newConfig}
	if err := tempOM.validateConfig(); err != nil {
		return fmt.Errorf("invalid new configuration: %w", err)
	}

	// Fechar writer atual se existir
	if om.fileWriter != nil {
		if err := om.fileWriter.Close(); err != nil {
			return fmt.Errorf("failed to close current file writer: %w", err)
		}
		om.fileWriter = nil
		om.isFileMode = false
	}

	// Atualizar configuração
	om.config = newConfig

	// Configurar novo writer se necessário
	if newConfig.FilePath != "" {
		if err := om.setupFileOutput(); err != nil {
			return fmt.Errorf("failed to setup new file output: %w", err)
		}
		om.isFileMode = true
	}

	return nil
}

// IsFileMode retorna true se o OutputManager está configurado para escrever em arquivo
func (om *OutputManager) IsFileMode() bool {
	return om.isFileMode
}

// GetFilePath retorna o caminho do arquivo de log atual
func (om *OutputManager) GetFilePath() string {
	return om.config.FilePath
}

// String retorna uma representação em string da configuração para debugging
func (oc OutputConfig) String() string {
	return fmt.Sprintf("OutputConfig{FilePath: %s, MaxSize: %d MB, MaxAge: %d days, MaxBackups: %d, Compress: %t, LocalTime: %t}",
		oc.FilePath, oc.MaxSize, oc.MaxAge, oc.MaxBackups, oc.Compress, oc.LocalTime)
}

// AddRotationHook adiciona um hook que será chamado quando ocorrer rotação
func (om *OutputManager) AddRotationHook(hook RotationHook) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.rotationHooks = append(om.rotationHooks, hook)
}

// RemoveAllRotationHooks remove todos os hooks de rotação
func (om *OutputManager) RemoveAllRotationHooks() {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.rotationHooks = nil
}

// triggerRotationHooks dispara todos os hooks de rotação registrados
func (om *OutputManager) triggerRotationHooks(event RotationEvent) {
	om.mu.RLock()
	hooks := make([]RotationHook, len(om.rotationHooks))
	copy(hooks, om.rotationHooks)
	om.mu.RUnlock()

	// Executar hooks em goroutines separadas para não bloquear
	for _, hook := range hooks {
		go func(h RotationHook) {
			defer func() {
				if r := recover(); r != nil {
					// Log do panic do hook, mas não interrompe o processo
					fmt.Fprintf(os.Stderr, "Rotation hook panic: %v\n", r)
				}
			}()
			h(event)
		}(hook)
	}
}

// GetRotationStats retorna estatísticas de rotação
func (om *OutputManager) GetRotationStats() (lastRotation time.Time, rotationCount int64) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.lastRotation, om.rotationCount
}

// RotateWithRecovery força a rotação com mecanismo de recuperação
func (om *OutputManager) RotateWithRecovery() error {
	if om.fileWriter == nil {
		return fmt.Errorf("no file writer configured")
	}

	om.mu.Lock()
	defer om.mu.Unlock()

	// Verificar se o writer é um lumberjack.Logger
	lj, ok := om.fileWriter.(*lumberjack.Logger)
	if !ok {
		return fmt.Errorf("file writer does not support rotation")
	}

	// Obter informações do arquivo atual antes da rotação
	var fileSize int64
	if stat, err := os.Stat(om.config.FilePath); err == nil {
		fileSize = stat.Size()
	}

	// Tentar rotação
	rotationTime := time.Now()
	err := lj.Rotate()

	// Criar evento de rotação
	event := RotationEvent{
		Timestamp: rotationTime,
		OldFile:   om.config.FilePath,
		NewFile:   om.config.FilePath, // lumberjack mantém o mesmo nome
		FileSize:  fileSize,
		Success:   err == nil,
		Error:     err,
	}

	// Atualizar estatísticas
	if err == nil {
		om.lastRotation = rotationTime
		om.rotationCount++
	}

	// Disparar hooks
	go om.triggerRotationHooks(event)

	// Se a rotação falhou, tentar recuperação
	if err != nil {
		if recoveryErr := om.attemptRecovery(); recoveryErr != nil {
			return fmt.Errorf("rotation failed and recovery failed: rotation error: %w, recovery error: %v", err, recoveryErr)
		}
		// Recuperação bem-sucedida, mas ainda retornamos o erro original da rotação
		return fmt.Errorf("rotation failed but recovery succeeded: %w", err)
	}

	return nil
}

// attemptRecovery tenta recuperar de uma falha de rotação
func (om *OutputManager) attemptRecovery() error {
	// Estratégia de recuperação: recriar o writer
	if om.fileWriter != nil {
		// Tentar fechar o writer atual (pode falhar, mas tentamos)
		om.fileWriter.Close()
	}

	// Recriar o writer
	return om.setupFileOutput()
}

// ForceRotationIfNeeded verifica se é necessário forçar rotação baseado no tamanho
func (om *OutputManager) ForceRotationIfNeeded() error {
	if !om.isFileMode || om.fileWriter == nil {
		return nil
	}

	// Verificar tamanho do arquivo atual
	stat, err := os.Stat(om.config.FilePath)
	if err != nil {
		// Se não conseguir obter stat, não é um erro crítico
		return nil
	}

	// Converter MaxSize de MB para bytes
	maxSizeBytes := int64(om.config.MaxSize) * 1024 * 1024

	// Se o arquivo excedeu o tamanho máximo, forçar rotação
	if stat.Size() >= maxSizeBytes {
		return om.RotateWithRecovery()
	}

	return nil
}

// GetCurrentFileSize retorna o tamanho atual do arquivo de log em bytes
func (om *OutputManager) GetCurrentFileSize() (int64, error) {
	if !om.isFileMode {
		return 0, fmt.Errorf("not in file mode")
	}

	stat, err := os.Stat(om.config.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file stats: %w", err)
	}

	return stat.Size(), nil
}
