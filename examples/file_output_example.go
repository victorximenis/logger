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

func fileOutputExample() {
	fmt.Println("=== Exemplo de Saída para Arquivo e Rotação ===")

	// Criar diretório temporário para os exemplos
	tempDir := filepath.Join(os.TempDir(), "logger_examples")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir) // Limpar após o exemplo

	ctx := context.Background()

	// Exemplo 1: Saída apenas para arquivo
	fmt.Println("1. Saída apenas para arquivo:")
	fileOnlyConfig := logger.Config{
		ServiceName:   "file-only-service",
		Environment:   "development",
		Output:        logger.OutputFile,
		LogLevel:      core.INFO,
		LogFilePath:   filepath.Join(tempDir, "file-only.log"),
		TenantID:      "tenant-001",
		PrettyPrint:   false,
		CallerEnabled: true,
	}

	if err := logger.Init(fileOnlyConfig); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	logger.Info(ctx).Str("tipo", "arquivo_apenas").Msg("Log escrito apenas no arquivo")
	logger.Warn(ctx).Str("status", "warning").Msg("Aviso importante")
	logger.Error(ctx).Str("codigo", "ERR001").Msg("Erro de exemplo")

	// Verificar se o arquivo foi criado
	if _, err := os.Stat(fileOnlyConfig.LogFilePath); err == nil {
		fmt.Printf("✓ Arquivo de log criado: %s\n", fileOnlyConfig.LogFilePath)

		// Mostrar conteúdo do arquivo
		content, _ := os.ReadFile(fileOnlyConfig.LogFilePath)
		fmt.Printf("Conteúdo do arquivo:\n%s\n", string(content))
	} else {
		fmt.Printf("✗ Arquivo de log não foi criado\n")
	}

	// Exemplo 2: Saída para stdout E arquivo (MultiWriter)
	fmt.Println("\n2. Saída para stdout E arquivo (MultiWriter):")
	multiConfig := logger.Config{
		ServiceName:   "multi-output-service",
		Environment:   "production",
		Output:        logger.OutputStdout | logger.OutputFile, // Ambos
		LogLevel:      core.DEBUG,
		LogFilePath:   filepath.Join(tempDir, "multi-output.log"),
		TenantID:      "tenant-002",
		PrettyPrint:   true,
		CallerEnabled: false,
	}

	if err := logger.Init(multiConfig); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	fmt.Println("Os logs abaixo aparecerão tanto no console quanto no arquivo:")
	logger.Debug(ctx).Str("tipo", "multi_output").Msg("Debug visível em ambos os destinos")
	logger.Info(ctx).Str("operacao", "processamento").Int("items", 42).Msg("Processamento concluído")
	logger.Error(ctx).Str("modulo", "database").Msg("Falha na conexão com banco")

	// Verificar arquivo multi-output
	if _, err := os.Stat(multiConfig.LogFilePath); err == nil {
		fmt.Printf("\n✓ Arquivo multi-output criado: %s\n", multiConfig.LogFilePath)
	}

	// Exemplo 3: Demonstração de rotação de logs
	fmt.Println("\n3. Demonstração de rotação de logs:")

	// Configurar um arquivo com tamanho pequeno para forçar rotação
	rotationConfig := logger.Config{
		ServiceName:   "rotation-service",
		Environment:   "test",
		Output:        logger.OutputFile,
		LogLevel:      core.INFO,
		LogFilePath:   filepath.Join(tempDir, "rotation.log"),
		TenantID:      "",
		PrettyPrint:   false,
		CallerEnabled: false,
	}

	if err := logger.Init(rotationConfig); err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		return
	}

	// Gerar muitos logs para demonstrar rotação
	fmt.Println("Gerando logs para demonstrar rotação...")
	for i := 0; i < 50; i++ {
		logger.Info(ctx).
			Int("iteration", i).
			Str("data", fmt.Sprintf("Esta é uma mensagem de log número %d com dados suficientes para ocupar espaço", i)).
			Str("timestamp", time.Now().Format(time.RFC3339)).
			Msg("Log de exemplo para rotação")

		// Pequena pausa para simular aplicação real
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("✓ Logs gerados no arquivo: %s\n", rotationConfig.LogFilePath)

	// Exemplo 4: Configuração via variáveis de ambiente
	fmt.Println("\n4. Configuração via variáveis de ambiente:")

	// Definir variáveis de ambiente para saída em arquivo
	os.Setenv("LOGGER_SERVICE_NAME", "env-file-service")
	os.Setenv("LOGGER_ENVIRONMENT", "staging")
	os.Setenv("LOGGER_OUTPUT", "file")
	os.Setenv("LOGGER_LOG_LEVEL", "warn")
	os.Setenv("LOGGER_LOG_FILE_PATH", filepath.Join(tempDir, "env-config.log"))
	os.Setenv("LOGGER_TENANT_ID", "tenant-env")
	os.Setenv("LOGGER_PRETTY_PRINT", "false")
	os.Setenv("LOGGER_CALLER_ENABLED", "true")

	// Inicializar com configuração do ambiente
	if err := logger.InitFromEnv(); err != nil {
		fmt.Printf("Erro ao inicializar logger do ambiente: %v\n", err)
		return
	}

	logger.Warn(ctx).Str("fonte", "ambiente").Msg("Logger configurado via variáveis de ambiente")
	logger.Error(ctx).Str("tipo", "critico").Msg("Erro crítico registrado")

	envLogPath := filepath.Join(tempDir, "env-config.log")
	if _, err := os.Stat(envLogPath); err == nil {
		fmt.Printf("✓ Arquivo de log do ambiente criado: %s\n", envLogPath)
	}

	// Exemplo 5: Demonstração de diferentes formatos
	fmt.Println("\n5. Comparação de formatos (Pretty vs JSON):")

	// Pretty print para desenvolvimento
	prettyConfig := logger.Config{
		ServiceName:   "pretty-service",
		Environment:   "development",
		Output:        logger.OutputStdout,
		LogLevel:      core.INFO,
		PrettyPrint:   true,
		CallerEnabled: true,
	}

	fmt.Println("Formato Pretty (desenvolvimento):")
	if err := logger.Init(prettyConfig); err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	logger.Info(ctx).Str("formato", "pretty").Bool("legivel", true).Msg("Log em formato legível")

	// JSON para produção
	jsonConfig := logger.Config{
		ServiceName:   "json-service",
		Environment:   "production",
		Output:        logger.OutputStdout,
		LogLevel:      core.INFO,
		PrettyPrint:   false,
		CallerEnabled: false,
	}

	fmt.Println("\nFormato JSON (produção):")
	if err := logger.Init(jsonConfig); err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}
	logger.Info(ctx).Str("formato", "json").Bool("estruturado", true).Msg("Log em formato JSON")

	// Listar arquivos criados
	fmt.Println("\n=== Arquivos de log criados ===")
	files, err := filepath.Glob(filepath.Join(tempDir, "*.log*"))
	if err == nil {
		for _, file := range files {
			info, _ := os.Stat(file)
			fmt.Printf("- %s (%d bytes)\n", filepath.Base(file), info.Size())
		}
	}

	fmt.Println("\n=== Exemplo Concluído ===")
	fmt.Printf("Arquivos temporários em: %s\n", tempDir)
	fmt.Println("(Os arquivos serão removidos automaticamente)")

	// Limpar variáveis de ambiente
	os.Unsetenv("LOGGER_SERVICE_NAME")
	os.Unsetenv("LOGGER_ENVIRONMENT")
	os.Unsetenv("LOGGER_OUTPUT")
	os.Unsetenv("LOGGER_LOG_LEVEL")
	os.Unsetenv("LOGGER_LOG_FILE_PATH")
	os.Unsetenv("LOGGER_TENANT_ID")
	os.Unsetenv("LOGGER_PRETTY_PRINT")
	os.Unsetenv("LOGGER_CALLER_ENABLED")
}
