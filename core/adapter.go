package core

import "context"

// Level representa os níveis de log disponíveis
type Level int

const (
	// DEBUG representa o nível de debug para informações detalhadas de depuração
	DEBUG Level = iota
	// INFO representa o nível de informação para mensagens informativas gerais
	INFO
	// WARN representa o nível de aviso para situações que merecem atenção
	WARN
	// ERROR representa o nível de erro para erros que não impedem a execução
	ERROR
	// FATAL representa o nível fatal para erros críticos que impedem a execução
	FATAL
)

// String retorna a representação em string do nível de log
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LoggerAdapter define a interface que deve ser implementada por diferentes
// bibliotecas de logging para integração com o sistema de logging unificado.
// Esta interface serve como uma ponte entre a interface pública Logger
// e as implementações concretas de logging.
type LoggerAdapter interface {
	// Log registra uma mensagem com o nível especificado, contexto e campos adicionais.
	// O contexto pode conter informações como trace ID, user ID, etc.
	// Os campos permitem adicionar metadados estruturados à entrada de log.
	Log(ctx context.Context, level Level, msg string, fields map[string]interface{})

	// WithContext retorna uma nova instância do adapter com o contexto especificado.
	// Isso permite que informações do contexto sejam propagadas através das chamadas de log.
	WithContext(ctx context.Context) LoggerAdapter

	// IsLevelEnabled verifica se o nível de log especificado está habilitado.
	// Isso permite otimizações evitando processamento desnecessário para logs
	// que não serão registrados.
	IsLevelEnabled(level Level) bool
}
