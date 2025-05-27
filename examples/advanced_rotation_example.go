package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/victorximenis/logger"
	"github.com/victorximenis/logger/core"
)

func advancedRotationExample() {
	fmt.Println("=== Exemplo de Rotação Avançada com Hooks e Monitoramento ===")

	// Criar diretório temporário para os exemplos
	tempDir := filepath.Join(os.TempDir(), "logger_advanced_rotation")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir) // Limpar após o exemplo

	ctx := context.Background()

	// Configurar logger com arquivo pequeno para demonstrar rotação
	config := logger.Config{
		ServiceName:   "advanced-rotation-service",
		Environment:   "development",
		Output:        logger.OutputFile,
		LogLevel:      core.INFO,
		LogFilePath:   filepath.Join(tempDir, "advanced.log"),
		TenantID:      "tenant-advanced",
		PrettyPrint:   false,
		CallerEnabled: false,
	}

	if err := logger.Init(config); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	// Obter o OutputManager para configurar hooks
	loggerInstance := logger.GetLogger()
	if loggerInstance == nil {
		fmt.Println("Erro: logger não inicializado")
		return
	}

	// Simular acesso ao OutputManager (em uma implementação real,
	// você teria acesso direto ao OutputManager)
	outputManager, err := core.NewOutputManager(core.NewOutputConfig(config.LogFilePath))
	if err != nil {
		fmt.Printf("Erro ao criar OutputManager: %v\n", err)
		return
	}
	defer outputManager.Close()

	fmt.Println("1. Configurando hooks de rotação:")

	// Hook 1: Notificação simples
	notificationHook := func(event core.RotationEvent) {
		if event.Success {
			fmt.Printf("✓ [HOOK] Rotação bem-sucedida em %s - Arquivo: %s (%d bytes)\n",
				event.Timestamp.Format("15:04:05"), event.OldFile, event.FileSize)
		} else {
			fmt.Printf("✗ [HOOK] Falha na rotação em %s - Erro: %v\n",
				event.Timestamp.Format("15:04:05"), event.Error)
		}
	}

	// Hook 2: Estatísticas detalhadas
	statsHook := func(event core.RotationEvent) {
		lastRotation, count := outputManager.GetRotationStats()
		fmt.Printf("📊 [STATS] Total de rotações: %d, Última rotação: %s\n",
			count, lastRotation.Format("15:04:05"))
	}

	// Hook 3: Simulação de backup/upload
	backupHook := func(event core.RotationEvent) {
		if event.Success {
			fmt.Printf("☁️  [BACKUP] Simulando upload do arquivo rotacionado para cloud storage...\n")
			time.Sleep(100 * time.Millisecond) // Simular operação de upload
			fmt.Printf("☁️  [BACKUP] Upload concluído com sucesso!\n")
		}
	}

	// Adicionar hooks
	outputManager.AddRotationHook(notificationHook)
	outputManager.AddRotationHook(statsHook)
	outputManager.AddRotationHook(backupHook)

	fmt.Println("✓ Hooks configurados: notificação, estatísticas e backup")

	fmt.Println("2. Demonstrando monitoramento de tamanho de arquivo:")

	// Gerar logs e monitorar tamanho
	for i := 0; i < 10; i++ {
		logger.Info(ctx).
			Int("iteration", i).
			Str("data", fmt.Sprintf("Log entry %d with some data to increase file size", i)).
			Str("timestamp", time.Now().Format(time.RFC3339)).
			Msg("Exemplo de log para demonstrar monitoramento")

		// Verificar tamanho atual
		if size, err := outputManager.GetCurrentFileSize(); err == nil {
			fmt.Printf("📏 Tamanho atual do arquivo: %d bytes\n", size)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("3. Demonstrando rotação manual com recuperação:")

	// Executar rotação manual
	fmt.Println("🔄 Executando rotação manual...")
	if err := outputManager.RotateWithRecovery(); err != nil {
		fmt.Printf("Erro na rotação: %v\n", err)
	} else {
		fmt.Println("✓ Rotação manual executada com sucesso")
	}

	// Aguardar hooks serem executados
	time.Sleep(200 * time.Millisecond)

	fmt.Println("\n4. Demonstrando verificação automática de rotação:")

	// Gerar mais logs para testar verificação automática
	for i := 0; i < 20; i++ {
		logger.Info(ctx).
			Int("batch", 2).
			Int("iteration", i).
			Str("large_data", fmt.Sprintf("Large log entry %d with substantial amount of data to trigger automatic rotation check", i)).
			Str("extra_field", "additional data to increase log size").
			Msg("Log para testar verificação automática de rotação")

		// Verificar se rotação é necessária
		if err := outputManager.ForceRotationIfNeeded(); err != nil {
			fmt.Printf("Erro na verificação de rotação: %v\n", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n5. Estatísticas finais:")

	lastRotation, totalRotations := outputManager.GetRotationStats()
	fmt.Printf("📈 Total de rotações executadas: %d\n", totalRotations)
	if !lastRotation.IsZero() {
		fmt.Printf("📅 Última rotação: %s\n", lastRotation.Format("15:04:05"))
	} else {
		fmt.Printf("📅 Nenhuma rotação executada\n")
	}

	if size, err := outputManager.GetCurrentFileSize(); err == nil {
		fmt.Printf("📏 Tamanho final do arquivo: %d bytes\n", size)
	}

	fmt.Println("\n6. Demonstrando remoção de hooks:")

	// Remover todos os hooks
	outputManager.RemoveAllRotationHooks()
	fmt.Println("🗑️  Todos os hooks removidos")

	// Executar rotação sem hooks
	fmt.Println("🔄 Executando rotação sem hooks...")
	if err := outputManager.Rotate(); err != nil {
		fmt.Printf("Erro na rotação: %v\n", err)
	} else {
		fmt.Println("✓ Rotação executada (sem notificações de hooks)")
	}

	// Aguardar um pouco para confirmar que não há hooks
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n7. Demonstrando configuração de rotação personalizada:")

	// Criar OutputManager com configuração personalizada
	customConfig := core.OutputConfig{
		FilePath:   filepath.Join(tempDir, "custom-rotation.log"),
		MaxSize:    5,    // 5MB
		MaxAge:     3,    // 3 dias
		MaxBackups: 10,   // 10 backups
		Compress:   true, // Comprimir arquivos rotacionados
		LocalTime:  true, // Usar horário local
	}

	customOM, err := core.NewOutputManager(customConfig)
	if err != nil {
		fmt.Printf("Erro ao criar OutputManager personalizado: %v\n", err)
		return
	}
	defer customOM.Close()

	fmt.Printf("⚙️  Configuração personalizada: %s\n", customConfig.String())

	// Adicionar hook personalizado
	customHook := func(event core.RotationEvent) {
		fmt.Printf("🎯 [CUSTOM] Rotação personalizada - Compressão: %t, Horário local: %t\n",
			customConfig.Compress, customConfig.LocalTime)
	}

	customOM.AddRotationHook(customHook)

	// Gerar alguns logs no OutputManager personalizado
	writer := customOM.GetWriter()
	for i := 0; i < 5; i++ {
		logLine := fmt.Sprintf("Custom log entry %d with timestamp %s\n", i, time.Now().Format(time.RFC3339))
		writer.Write([]byte(logLine))
	}

	fmt.Println("✓ Logs escritos no OutputManager personalizado")

	// Listar arquivos criados
	fmt.Println("\n=== Arquivos de log criados ===")
	files, err := filepath.Glob(filepath.Join(tempDir, "*.log*"))
	if err == nil {
		for _, file := range files {
			info, _ := os.Stat(file)
			fmt.Printf("- %s (%d bytes)\n", filepath.Base(file), info.Size())
		}
	}

	fmt.Println("\n=== Exemplo de Rotação Avançada Concluído ===")
	fmt.Printf("Arquivos temporários em: %s\n", tempDir)
	fmt.Println("(Os arquivos serão removidos automaticamente)")
}
