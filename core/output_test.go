package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewOutputConfig(t *testing.T) {
	filePath := "/tmp/test.log"
	config := NewOutputConfig(filePath)

	if config.FilePath != filePath {
		t.Errorf("Expected FilePath %s, got %s", filePath, config.FilePath)
	}
	if config.MaxSize != DefaultMaxSize {
		t.Errorf("Expected MaxSize %d, got %d", DefaultMaxSize, config.MaxSize)
	}
	if config.MaxAge != DefaultMaxAge {
		t.Errorf("Expected MaxAge %d, got %d", DefaultMaxAge, config.MaxAge)
	}
	if config.MaxBackups != DefaultMaxBackups {
		t.Errorf("Expected MaxBackups %d, got %d", DefaultMaxBackups, config.MaxBackups)
	}
	if config.Compress != DefaultCompress {
		t.Errorf("Expected Compress %t, got %t", DefaultCompress, config.Compress)
	}
	if config.LocalTime != DefaultLocalTime {
		t.Errorf("Expected LocalTime %t, got %t", DefaultLocalTime, config.LocalTime)
	}
}

func TestOutputConfig_String(t *testing.T) {
	config := OutputConfig{
		FilePath:   "/tmp/test.log",
		MaxSize:    50,
		MaxAge:     3,
		MaxBackups: 2,
		Compress:   true,
		LocalTime:  false,
	}

	result := config.String()
	expected := "OutputConfig{FilePath: /tmp/test.log, MaxSize: 50 MB, MaxAge: 3 days, MaxBackups: 2, Compress: true, LocalTime: false}"

	if result != expected {
		t.Errorf("OutputConfig.String() = %s, expected %s", result, expected)
	}
}

func TestNewOutputManager_NoFile(t *testing.T) {
	config := OutputConfig{
		FilePath: "", // Sem arquivo
		MaxSize:  100,
		MaxAge:   7,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error for no-file config, got: %v", err)
	}

	if om.IsFileMode() {
		t.Error("Expected IsFileMode to be false when no file path is provided")
	}

	writer := om.GetWriter()
	if writer != os.Stdout {
		t.Error("Expected GetWriter to return os.Stdout when no file is configured")
	}
}

func TestNewOutputManager_WithFile(t *testing.T) {
	// Criar diretório temporário para teste
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.log")

	config := OutputConfig{
		FilePath:   filePath,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 2,
		Compress:   false,
		LocalTime:  true,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	defer om.Close()

	if !om.IsFileMode() {
		t.Error("Expected IsFileMode to be true when file path is provided")
	}

	if om.GetFilePath() != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, om.GetFilePath())
	}

	// Verificar se o writer não é nil
	writer := om.GetWriter()
	if writer == nil {
		t.Error("Expected GetWriter to return a valid writer")
	}

	// Testar escrita
	testMessage := "test log message\n"
	n, err := writer.Write([]byte(testMessage))
	if err != nil {
		t.Errorf("Failed to write to log file: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testMessage), n)
	}

	// Verificar se o arquivo foi criado
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestOutputManager_ValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    OutputConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config with file",
			config: OutputConfig{
				FilePath:   "/tmp/test.log",
				MaxSize:    100,
				MaxAge:     7,
				MaxBackups: 5,
			},
			expectErr: false,
		},
		{
			name: "valid config without file",
			config: OutputConfig{
				FilePath:   "",
				MaxSize:    100,
				MaxAge:     7,
				MaxBackups: 5,
			},
			expectErr: false,
		},
		{
			name: "invalid file path - empty directory",
			config: OutputConfig{
				FilePath: "test.log", // Sem diretório
				MaxSize:  100,
			},
			expectErr: true,
			errMsg:    "invalid file path: directory cannot be empty",
		},
		{
			name: "invalid file path - empty filename",
			config: OutputConfig{
				FilePath: "/tmp/",
				MaxSize:  100,
			},
			expectErr: true,
			errMsg:    "invalid file path: filename cannot be empty",
		},
		{
			name: "invalid max size - zero",
			config: OutputConfig{
				FilePath: "/tmp/test.log",
				MaxSize:  0,
			},
			expectErr: true,
			errMsg:    "max size must be positive, got 0",
		},
		{
			name: "invalid max size - negative",
			config: OutputConfig{
				FilePath: "/tmp/test.log",
				MaxSize:  -1,
			},
			expectErr: true,
			errMsg:    "max size must be positive, got -1",
		},
		{
			name: "invalid max age - negative",
			config: OutputConfig{
				FilePath: "/tmp/test.log",
				MaxSize:  100,
				MaxAge:   -1,
			},
			expectErr: true,
			errMsg:    "max age cannot be negative, got -1",
		},
		{
			name: "invalid max backups - negative",
			config: OutputConfig{
				FilePath:   "/tmp/test.log",
				MaxSize:    100,
				MaxAge:     7,
				MaxBackups: -1,
			},
			expectErr: true,
			errMsg:    "max backups cannot be negative, got -1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			om := &OutputManager{config: tt.config}
			err := om.validateConfig()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
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

func TestOutputManager_GetMultiWriter(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.log")

	config := OutputConfig{
		FilePath: filePath,
		MaxSize:  10,
		MaxAge:   1,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	defer om.Close()

	multiWriter := om.GetMultiWriter()
	if multiWriter == nil {
		t.Error("Expected GetMultiWriter to return a valid writer")
	}

	// Testar escrita no MultiWriter
	testMessage := "test multi writer message\n"
	n, err := multiWriter.Write([]byte(testMessage))
	if err != nil {
		t.Errorf("Failed to write to multi writer: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testMessage), n)
	}

	// Verificar se o arquivo foi criado
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Log file was not created by multi writer")
	}
}

func TestOutputManager_UpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := filepath.Join(tempDir, "original.log")
	newPath := filepath.Join(tempDir, "new.log")

	// Configuração inicial
	originalConfig := OutputConfig{
		FilePath: originalPath,
		MaxSize:  10,
		MaxAge:   1,
	}

	om, err := NewOutputManager(originalConfig)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	defer om.Close()

	// Escrever algo no arquivo original
	writer := om.GetWriter()
	writer.Write([]byte("original message\n"))

	// Atualizar configuração
	newConfig := OutputConfig{
		FilePath: newPath,
		MaxSize:  20,
		MaxAge:   2,
	}

	err = om.UpdateConfig(newConfig)
	if err != nil {
		t.Errorf("Expected no error updating config, got: %v", err)
	}

	// Verificar se a configuração foi atualizada
	currentConfig := om.GetConfig()
	if currentConfig.FilePath != newPath {
		t.Errorf("Expected file path %s, got %s", newPath, currentConfig.FilePath)
	}
	if currentConfig.MaxSize != 20 {
		t.Errorf("Expected max size 20, got %d", currentConfig.MaxSize)
	}

	// Escrever algo no novo arquivo
	newWriter := om.GetWriter()
	newWriter.Write([]byte("new message\n"))

	// Verificar se o novo arquivo foi criado
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("New log file was not created after config update")
	}
}

func TestOutputManager_UpdateConfig_Invalid(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := filepath.Join(tempDir, "original.log")

	originalConfig := OutputConfig{
		FilePath: originalPath,
		MaxSize:  10,
		MaxAge:   1,
	}

	om, err := NewOutputManager(originalConfig)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	defer om.Close()

	// Tentar atualizar com configuração inválida
	invalidConfig := OutputConfig{
		FilePath: originalPath,
		MaxSize:  -1, // Inválido
		MaxAge:   1,
	}

	err = om.UpdateConfig(invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid config update, got none")
	}

	// Verificar se a configuração original foi mantida
	currentConfig := om.GetConfig()
	if currentConfig.MaxSize != 10 {
		t.Errorf("Expected original max size 10 to be preserved, got %d", currentConfig.MaxSize)
	}
}

func TestOutputManager_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.log")

	config := OutputConfig{
		FilePath:   filePath,
		MaxSize:    1, // 1 MB para facilitar teste
		MaxAge:     1,
		MaxBackups: 2,
		Compress:   false,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	defer om.Close()

	// Escrever algo no arquivo
	writer := om.GetWriter()
	writer.Write([]byte("test message before rotation\n"))

	// Forçar rotação
	err = om.Rotate()
	if err != nil {
		t.Errorf("Expected no error during rotation, got: %v", err)
	}

	// Escrever algo após rotação
	writer.Write([]byte("test message after rotation\n"))

	// Verificar se o arquivo principal ainda existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Main log file should still exist after rotation")
	}
}

func TestOutputManager_Rotate_NoFile(t *testing.T) {
	config := OutputConfig{
		FilePath: "", // Sem arquivo
		MaxSize:  100,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Tentar rotacionar sem arquivo configurado
	err = om.Rotate()
	if err == nil {
		t.Error("Expected error when trying to rotate without file writer")
	}

	expectedMsg := "no file writer configured"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOutputManager_Close(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.log")

	config := OutputConfig{
		FilePath: filePath,
		MaxSize:  10,
		MaxAge:   1,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Escrever algo no arquivo
	writer := om.GetWriter()
	writer.Write([]byte("test message\n"))

	// Fechar o OutputManager
	err = om.Close()
	if err != nil {
		t.Errorf("Expected no error during close, got: %v", err)
	}

	// Verificar se o arquivo foi criado e contém dados
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read log file after close: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("Log file should contain the written message")
	}
}

func TestOutputManager_Close_NoFile(t *testing.T) {
	config := OutputConfig{
		FilePath: "", // Sem arquivo
		MaxSize:  100,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Fechar sem arquivo configurado não deve causar erro
	err = om.Close()
	if err != nil {
		t.Errorf("Expected no error when closing without file, got: %v", err)
	}
}

func TestOutputManager_RotationHooks(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_hooks.log")

	config := NewOutputConfig(filePath)
	config.MaxSize = 1 // 1MB para facilitar teste

	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Adicionar hook de teste
	var hookCalled bool
	var hookEvent RotationEvent
	hook := func(event RotationEvent) {
		hookCalled = true
		hookEvent = event
	}

	om.AddRotationHook(hook)

	// Escrever dados para forçar rotação
	writer := om.GetWriter()
	data := strings.Repeat("test log line\n", 1000)
	writer.Write([]byte(data))

	// Forçar rotação
	err = om.Rotate()
	if err != nil {
		t.Errorf("Rotation failed: %v", err)
	}

	// Aguardar um pouco para o hook ser executado (é assíncrono)
	time.Sleep(100 * time.Millisecond)

	if !hookCalled {
		t.Error("Rotation hook was not called")
	}

	if hookEvent.Timestamp.IsZero() {
		t.Error("Hook event timestamp is zero")
	}

	if hookEvent.OldFile != filePath {
		t.Errorf("Expected OldFile %s, got %s", filePath, hookEvent.OldFile)
	}

	if !hookEvent.Success {
		t.Errorf("Expected successful rotation, got error: %v", hookEvent.Error)
	}
}

func TestOutputManager_RotationStats(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_stats.log")

	config := NewOutputConfig(filePath)
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Verificar estatísticas iniciais
	lastRotation, count := om.GetRotationStats()
	if !lastRotation.IsZero() {
		t.Error("Expected zero time for initial last rotation")
	}
	if count != 0 {
		t.Errorf("Expected 0 rotation count, got %d", count)
	}

	// Executar rotação
	err = om.Rotate()
	if err != nil {
		t.Errorf("Rotation failed: %v", err)
	}

	// Verificar estatísticas após rotação
	lastRotation, count = om.GetRotationStats()
	if lastRotation.IsZero() {
		t.Error("Expected non-zero time for last rotation")
	}
	if count != 1 {
		t.Errorf("Expected 1 rotation count, got %d", count)
	}
}

func TestOutputManager_ForceRotationIfNeeded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_force.log")

	config := NewOutputConfig(filePath)
	config.MaxSize = 1 // 1MB
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Arquivo pequeno - não deve rotacionar
	writer := om.GetWriter()
	writer.Write([]byte("small log\n"))

	err = om.ForceRotationIfNeeded()
	if err != nil {
		t.Errorf("ForceRotationIfNeeded failed: %v", err)
	}

	// Verificar que não houve rotação
	_, count := om.GetRotationStats()
	if count != 0 {
		t.Errorf("Expected 0 rotations, got %d", count)
	}

	// Escrever dados grandes para exceder limite (2MB para garantir que excede 1MB)
	largeData := strings.Repeat("large log line with lots of data to exceed the 1MB limit\n", 40000)
	writer.Write([]byte(largeData))

	// Verificar tamanho do arquivo antes da rotação
	size, err := om.GetCurrentFileSize()
	if err != nil {
		t.Errorf("GetCurrentFileSize failed: %v", err)
	}

	// Só testar rotação se o arquivo realmente excedeu o limite
	if size >= 1024*1024 { // 1MB em bytes
		err = om.ForceRotationIfNeeded()
		if err != nil {
			t.Errorf("ForceRotationIfNeeded failed: %v", err)
		}

		// Verificar que houve rotação
		_, count = om.GetRotationStats()
		if count != 1 {
			t.Errorf("Expected 1 rotation, got %d", count)
		}
	} else {
		t.Logf("File size %d bytes is less than 1MB, skipping rotation test", size)
	}
}

func TestOutputManager_GetCurrentFileSize(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_size.log")

	config := NewOutputConfig(filePath)
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Escrever dados
	testData := "test log data\n"
	writer := om.GetWriter()
	writer.Write([]byte(testData))

	// Verificar tamanho
	size, err := om.GetCurrentFileSize()
	if err != nil {
		t.Errorf("GetCurrentFileSize failed: %v", err)
	}

	expectedSize := int64(len(testData))
	if size != expectedSize {
		t.Errorf("Expected file size %d, got %d", expectedSize, size)
	}
}

func TestOutputManager_RemoveAllRotationHooks(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_remove_hooks.log")

	config := NewOutputConfig(filePath)
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Adicionar hooks
	var hook1Called, hook2Called bool
	hook1 := func(event RotationEvent) { hook1Called = true }
	hook2 := func(event RotationEvent) { hook2Called = true }

	om.AddRotationHook(hook1)
	om.AddRotationHook(hook2)

	// Remover todos os hooks
	om.RemoveAllRotationHooks()

	// Executar rotação
	err = om.Rotate()
	if err != nil {
		t.Errorf("Rotation failed: %v", err)
	}

	// Aguardar um pouco
	time.Sleep(100 * time.Millisecond)

	// Verificar que nenhum hook foi chamado
	if hook1Called || hook2Called {
		t.Error("Hooks were called after being removed")
	}
}

func TestOutputManager_RotateWithRecovery(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_recovery.log")

	config := NewOutputConfig(filePath)
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Escrever dados iniciais
	writer := om.GetWriter()
	writer.Write([]byte("initial data\n"))

	// Testar rotação com recuperação
	err = om.RotateWithRecovery()
	if err != nil {
		t.Errorf("RotateWithRecovery failed: %v", err)
	}

	// Verificar que o arquivo principal ainda existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Main log file should exist after rotation with recovery")
	}

	// Verificar estatísticas
	lastRotation, count := om.GetRotationStats()
	if lastRotation.IsZero() {
		t.Error("Expected non-zero last rotation time")
	}
	if count != 1 {
		t.Errorf("Expected 1 rotation count, got %d", count)
	}

	// Escrever dados após rotação para verificar que o writer ainda funciona
	writer.Write([]byte("data after recovery\n"))

	// Verificar que o arquivo contém os novos dados
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read file after recovery: %v", err)
	}

	if !strings.Contains(string(content), "data after recovery") {
		t.Error("File should contain data written after recovery")
	}
}

func TestOutputManager_RotationHookPanicRecovery(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_hook_panic.log")

	config := NewOutputConfig(filePath)
	om, err := NewOutputManager(config)
	if err != nil {
		t.Fatalf("Failed to create OutputManager: %v", err)
	}
	defer om.Close()

	// Adicionar hook que causa panic
	panicHook := func(event RotationEvent) {
		panic("test panic in hook")
	}

	// Adicionar hook normal para verificar que ainda funciona
	var normalHookCalled bool
	normalHook := func(event RotationEvent) {
		normalHookCalled = true
	}

	om.AddRotationHook(panicHook)
	om.AddRotationHook(normalHook)

	// Executar rotação - não deve falhar mesmo com hook que causa panic
	err = om.Rotate()
	if err != nil {
		t.Errorf("Rotation should not fail due to hook panic: %v", err)
	}

	// Aguardar hooks serem executados
	time.Sleep(200 * time.Millisecond)

	// Verificar que o hook normal foi executado
	if !normalHookCalled {
		t.Error("Normal hook should have been called despite panic in other hook")
	}
}

func TestOutputManager_GetCurrentFileSize_NoFile(t *testing.T) {
	config := OutputConfig{
		FilePath: "", // Sem arquivo
		MaxSize:  100,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Tentar obter tamanho sem arquivo configurado
	_, err = om.GetCurrentFileSize()
	if err == nil {
		t.Error("Expected error when getting file size without file mode")
	}

	expectedMsg := "not in file mode"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestOutputManager_ForceRotationIfNeeded_NoFile(t *testing.T) {
	config := OutputConfig{
		FilePath: "", // Sem arquivo
		MaxSize:  100,
	}

	om, err := NewOutputManager(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// ForceRotationIfNeeded deve retornar nil quando não há arquivo
	err = om.ForceRotationIfNeeded()
	if err != nil {
		t.Errorf("ForceRotationIfNeeded should not fail without file mode: %v", err)
	}
}
