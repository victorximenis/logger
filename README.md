# Logger - Sistema de Logging Unificado para Go

Um pacote de logging estruturado e flexível para Go que utiliza o padrão Adapter para permitir integração com diferentes bibliotecas de logging, com suporte completo a observabilidade e conformidade LGPD.

## Características

- **Interface fluente** com method chaining para construção de logs
- **Logging estruturado** com campos tipados
- **Padrão Adapter** para integração com diferentes bibliotecas de logging
- **Níveis de log padrão** (DEBUG, INFO, WARN, ERROR, FATAL)
- **Propagação de contexto** para rastreamento de requisições
- **Campos pré-definidos** por instância do logger
- **Otimização de performance** com verificação de nível habilitado
- **Observabilidade integrada** com Datadog e ELK Stack
- **Middlewares HTTP** para Gin, Fiber e Chi
- **Integração com PostgreSQL** via PGX
- **Sanitização LGPD** automática de dados sensíveis
- **Correlation IDs** automáticos para rastreamento
- **Distributed tracing** com Datadog
- **Métricas automáticas** e dashboards
- **Adapter Zerolog** incluído para uso imediato

## Instalação

```bash
go get github.com/victorximenis/logger
```

## Uso Básico

### 1. Inicialização Rápida com Perfis

```go
package main

import (
    "context"
    "github.com/victorximenis/logger"
)

func main() {
    // Inicialização para produção (Datadog + ELK habilitados)
    err := logger.InitWithProfile("production", "my-service")
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    
    // Usar o logger global
    logger.Info(ctx).
        Str("user_id", "123").
        Int("attempt", 1).
        Msg("User login successful")
}
```

### 2. Configuração Manual com Observabilidade

```go
package main

import (
    "context"
    "github.com/victorximenis/logger"
)

func main() {
    // Configuração personalizada
    config := logger.NewConfig()
    config.ServiceName = "auth-service"
    config.Environment = "production"
    config.LogLevel = logger.INFO
    
    // Habilitar observabilidade
    config.Observability.Enabled = true
    config.Observability.EnableDatadog = true
    config.Observability.EnableELK = true
    config.Observability.EnableCorrelationID = true
    
    err := logger.Init(config)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    logger.Info(ctx).Msg("Service started")
}
```

### 3. Usando Middlewares HTTP

#### Gin

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/middlewares"
)

func main() {
    logger.InitWithProfile("production", "api-service")
    
    r := gin.New()
    
    // Adicionar middleware de logging
    r.Use(middlewares.GinLogger())
    
    r.GET("/users/:id", func(c *gin.Context) {
        // O contexto já contém correlation_id, request_id, etc.
        logger.Info(c.Request.Context()).
            Str("user_id", c.Param("id")).
            Msg("User requested")
        
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    r.Run(":8080")
}
```

#### Fiber

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/middlewares"
)

func main() {
    logger.InitWithProfile("production", "api-service")
    
    app := fiber.New()
    
    // Adicionar middleware de logging
    app.Use(middlewares.FiberLogger())
    
    app.Get("/users/:id", func(c *fiber.Ctx) error {
        // O contexto já contém correlation_id, request_id, etc.
        logger.Info(c.Context()).
            Str("user_id", c.Params("id")).
            Msg("User requested")
        
        return c.JSON(fiber.Map{"status": "ok"})
    })
    
    app.Listen(":8080")
}
```

#### Chi

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/middlewares"
)

func main() {
    logger.InitWithProfile("production", "api-service")
    
    r := chi.NewRouter()
    
    // Adicionar middleware de logging
    r.Use(middlewares.ChiLogger())
    
    r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
        // O contexto já contém correlation_id, request_id, etc.
        logger.Info(r.Context()).
            Str("user_id", chi.URLParam(r, "id")).
            Msg("User requested")
        
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status": "ok"}`))
    })
    
    http.ListenAndServe(":8080", r)
}
```

### 4. Integração com PostgreSQL (PGX)

```go
package main

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/integrations"
)

func main() {
    logger.InitWithProfile("production", "db-service")
    
    // Configurar pool de conexões com logging
    config, err := pgxpool.ParseConfig("postgres://user:pass@localhost/db")
    if err != nil {
        panic(err)
    }
    
    // Adicionar tracer de logging
    config.ConnConfig.Tracer = integrations.NewPGXTracer()
    
    pool, err := pgxpool.NewWithConfig(context.Background(), config)
    if err != nil {
        panic(err)
    }
    defer pool.Close()
    
    // Todas as queries serão automaticamente logadas
    rows, err := pool.Query(context.Background(), "SELECT id, name FROM users WHERE active = $1", true)
    if err != nil {
        logger.Error(context.Background()).Err(err).Msg("Query failed")
        return
    }
    defer rows.Close()
    
    logger.Info(context.Background()).Msg("Query executed successfully")
}
```

### 5. Sanitização LGPD Automática

```go
package main

import (
    "context"
    "github.com/victorximenis/logger"
)

func main() {
    logger.InitWithProfile("production", "user-service")
    
    ctx := context.Background()
    
    // Dados sensíveis são automaticamente sanitizados
    logger.Info(ctx).
        Str("email", "user@example.com").           // Será sanitizado: u***@e***.com
        Str("cpf", "123.456.789-00").              // Será sanitizado: 123.***.***-**
        Str("phone", "+55 11 99999-9999").         // Será sanitizado: +55 11 9****-****
        Str("credit_card", "4111 1111 1111 1111"). // Será sanitizado: 4111 **** **** ****
        Str("password", "mypassword").             // Será sanitizado: [REDACTED]
        Msg("User data processed")
}
```

### 6. Correlation IDs e Distributed Tracing

```go
package main

import (
    "context"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/observability"
)

func main() {
    logger.InitWithProfile("production", "order-service")
    
    ctx := context.Background()
    
    // Adicionar correlation ID manualmente
    ctx = observability.ContextWithCorrelationID(ctx, "order-123")
    
    // Iniciar span do Datadog
    span := observability.StartSpan("process.order")
    defer span.Finish()
    ctx = observability.ContextWithSpan(ctx, span)
    
    // Logs incluirão automaticamente trace_id, span_id, correlation_id
    logger.Info(ctx).
        Str("order_id", "123").
        Float64("amount", 99.99).
        Msg("Processing order")
    
    // Registrar métricas
    observability.IncrementCounter("orders.processed", []string{"status:success"})
    observability.RecordDuration("orders.processing_time", span.Duration(), []string{"service:order"})
}
```

## Configuração de Observabilidade

### Variáveis de Ambiente

#### Configurações Gerais
```bash
# Habilitar observabilidade
LOGGER_OBSERVABILITY_ENABLED=true
OBSERVABILITY_ENABLED=true
OBSERVABILITY_DATADOG=true
OBSERVABILITY_ELK=true
OBSERVABILITY_CORRELATION_ID=true
```

#### Datadog
```bash
# Configurações básicas
DD_ENABLED=true
DD_SERVICE=my-service
DD_ENV=production
DD_VERSION=1.0.0
DD_AGENT_HOST=localhost:8126

# Tracing e métricas
DD_TRACING_ENABLED=true
DD_METRICS_ENABLED=true
DD_TRACE_SAMPLE_RATE=0.1

# Tags globais
DD_TAGS=team:backend,region:us-east-1
```

#### ELK Stack
```bash
# Configurações básicas
ELK_ENABLED=true
ELK_SERVICE=my-service
ELK_ENV=production
ELK_SERVICE_VERSION=1.0.0

# Elasticsearch
ELK_INDEX_PREFIX=logs
ELK_ECS_MAPPING=true
ELK_HOSTNAME=my-host
ELK_DATACENTER=us-east-1

# Campos personalizados
ELK_CUSTOM_FIELDS=team=backend,region=us-east-1
```

### Perfis de Configuração

#### Produção
```go
// Configuração automática para produção
config := logger.NewProductionConfig("my-service")
// - LogLevel: INFO
// - PrettyPrint: false
// - CallerEnabled: false
// - Datadog: habilitado (sampling 10%)
// - ELK: habilitado (ECS mapping)
// - CorrelationID: habilitado
```

#### Desenvolvimento
```go
// Configuração automática para desenvolvimento
config := logger.NewDevelopmentConfig("my-service")
// - LogLevel: DEBUG
// - PrettyPrint: true
// - CallerEnabled: true
// - Datadog: desabilitado
// - ELK: habilitado (formato simples)
// - CorrelationID: habilitado
```

#### Staging
```go
// Configuração automática para staging
config := logger.NewStagingConfig("my-service")
// - LogLevel: DEBUG
// - PrettyPrint: false
// - CallerEnabled: true
// - Datadog: habilitado
// - ELK: habilitado (ECS mapping)
// - CorrelationID: habilitado
```

## Adapter Zerolog Avançado

### Configuração Completa

```go
package main

import (
    "os"
    "github.com/victorximenis/logger"
    "github.com/victorximenis/logger/adapters"
    "github.com/victorximenis/logger/core"
)

func main() {
    // Configuração avançada do Zerolog
    config := &adapters.ZerologConfig{
        Writer:        os.Stdout,
        Level:         core.INFO,
        TimeFormat:    "", // Unix timestamp
        PrettyPrint:   false,
        CallerEnabled: false,
    }
    
    // Criar adapter e logger
    adapter := adapters.NewZerologAdapter(config)
    log := logger.New(adapter)
    
    // Logger com campos pré-definidos
    serviceLogger := log.WithFields(map[string]interface{}{
        "service": "auth",
        "version": "1.0.0",
    })
    
    ctx := context.Background()
    serviceLogger.Info(ctx).
        Str("user_id", "123").
        Int("attempt", 1).
        Msg("User login successful")
}
```

### Diferentes Tipos de Campos

```go
log.Info(ctx).
    Str("string_field", "value").           // Campo string
    Int("int_field", 42).                   // Campo inteiro
    Float64("float_field", 3.14).           // Campo float64
    Bool("bool_field", true).               // Campo booleano
    Err(errors.New("example error")).       // Campo de erro
    Any("any_field", customStruct).         // Campo de qualquer tipo
    Fields(map[string]interface{}{          // Múltiplos campos
        "key1": "value1",
        "key2": "value2",
    }).
    Msg("Log with various field types")
```

### Formatação de Mensagens

```go
// Mensagem simples
log.Info(ctx).Msg("Simple message")

// Mensagem formatada
log.Info(ctx).Msgf("User %s has %d points", "John", 100)

// Apenas campos sem mensagem
log.Info(ctx).Str("event", "user_action").Send()
```

## Níveis de Log

O pacote suporta os seguintes níveis de log:

- `DEBUG`: Informações detalhadas de depuração
- `INFO`: Mensagens informativas gerais
- `WARN`: Situações que merecem atenção
- `ERROR`: Erros que não impedem a execução
- `FATAL`: Erros críticos que impedem a execução

## Arquitetura

### Interfaces Principais

#### LoggerAdapter
Interface que deve ser implementada por diferentes bibliotecas de logging:

```go
type LoggerAdapter interface {
    Log(ctx context.Context, level Level, msg string, fields map[string]interface{})
    WithContext(ctx context.Context) LoggerAdapter
    IsLevelEnabled(level Level) bool
}
```

#### Logger
Interface pública para operações de logging:

```go
type Logger interface {
    Debug(ctx context.Context) LogEvent
    Info(ctx context.Context) LogEvent
    Warn(ctx context.Context) LogEvent
    Error(ctx context.Context) LogEvent
    Fatal(ctx context.Context) LogEvent
    WithContext(ctx context.Context) Logger
    WithFields(fields map[string]interface{}) Logger
}
```

#### LogEvent
Interface para construção fluente de entradas de log:

```go
type LogEvent interface {
    Str(key, val string) LogEvent
    Int(key string, val int) LogEvent
    Float64(key string, val float64) LogEvent
    Bool(key string, val bool) LogEvent
    Err(err error) LogEvent
    Any(key string, val interface{}) LogEvent
    Fields(fields map[string]interface{}) LogEvent
    Msg(msg string)
    Msgf(format string, args ...interface{})
    Send()
}
```

## Componentes Disponíveis

### Adapters
- **ZerologAdapter**: Implementação completa usando [zerolog](https://github.com/rs/zerolog)
- **DatadogLoggerAdapter**: Wrapper para integração com Datadog
- **ELKLoggerAdapter**: Wrapper para integração com ELK Stack
- **MultiObservabilityAdapter**: Combina múltiplos adapters
- **CorrelationIDAdapter**: Adiciona correlation IDs automáticos

### Middlewares HTTP
- **GinLogger**: Middleware para framework Gin
- **FiberLogger**: Middleware para framework Fiber
- **ChiLogger**: Middleware para framework Chi

### Integrações
- **PGXTracer**: Tracer para logging de queries PostgreSQL via PGX

### Sanitização
- **LGPDSanitizer**: Sanitização automática de dados sensíveis conforme LGPD

### Observabilidade
- **Datadog**: Distributed tracing, métricas e dashboards
- **ELK Stack**: Logs estruturados com Elastic Common Schema (ECS)
- **Correlation IDs**: Rastreamento de requisições entre serviços

## Exemplos de Adapters para Outras Bibliotecas

### Adapter para Logrus

```go
import "github.com/sirupsen/logrus"

type LogrusAdapter struct {
    logger *logrus.Logger
}

func (l *LogrusAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
    entry := l.logger.WithFields(logrus.Fields(fields))
    
    switch level {
    case core.DEBUG:
        entry.Debug(msg)
    case core.INFO:
        entry.Info(msg)
    case core.WARN:
        entry.Warn(msg)
    case core.ERROR:
        entry.Error(msg)
    case core.FATAL:
        entry.Fatal(msg)
    }
}
```

### Adapter para Zap

```go
import "go.uber.org/zap"

type ZapAdapter struct {
    logger *zap.Logger
}

func (z *ZapAdapter) Log(ctx context.Context, level core.Level, msg string, fields map[string]interface{}) {
    zapFields := make([]zap.Field, 0, len(fields))
    for k, v := range fields {
        zapFields = append(zapFields, zap.Any(k, v))
    }
    
    switch level {
    case core.DEBUG:
        z.logger.Debug(msg, zapFields...)
    case core.INFO:
        z.logger.Info(msg, zapFields...)
    // ... outros níveis
    }
}
```

## Monitoramento e Métricas

### Dashboards Datadog

O sistema automaticamente envia métricas para Datadog:

- `logger.log_count`: Contador de logs por nível
- `logger.error_count`: Contador de erros
- `http.request.duration`: Duração de requisições HTTP
- `http.request.count`: Contador de requisições HTTP
- `db.query.duration`: Duração de queries de banco
- `db.query.count`: Contador de queries de banco

### Logs Estruturados ELK

Logs são enviados para Elasticsearch seguindo o padrão ECS:

```json
{
  "@timestamp": "2024-01-15T10:30:00.000Z",
  "ecs.version": "8.0",
  "message": "User login successful",
  "log.level": "info",
  "service.name": "auth-service",
  "service.version": "1.0.0",
  "service.environment": "production",
  "host.name": "api-server-01",
  "user.id": "123",
  "trace.id": "abc123",
  "span.id": "def456",
  "http.request.method": "POST",
  "http.response.status_code": 200,
  "event.duration": 150000000
}
```

## Testes

Execute os testes com:

```bash
go test ./...
```

Para executar com cobertura:

```bash
go test ./... -cover
```

Para executar exemplos:

```bash
go run examples/basic_usage.go
go run examples/middleware_example.go
go run examples/observability_example.go
```

## Estrutura do Projeto

```
logger/
├── logger/
│   ├── core/              # Interfaces principais e tipos
│   │   ├── adapter.go     # Interface LoggerAdapter
│   │   ├── event.go       # Interface LogEvent
│   │   ├── level.go       # Níveis de log
│   │   └── output.go      # Gerenciamento de saída
│   ├── adapters/          # Implementações de adapters
│   │   └── zerolog.go     # Adapter para zerolog
│   ├── middlewares/       # Middlewares HTTP
│   │   ├── gin.go         # Middleware para Gin
│   │   ├── fiber.go       # Middleware para Fiber
│   │   └── chi.go         # Middleware para Chi
│   ├── integrations/      # Integrações com bancos de dados
│   │   └── pgx.go         # Integração com PostgreSQL via PGX
│   ├── observability/     # Sistema de observabilidade
│   │   ├── adapter.go     # Adapters de observabilidade
│   │   ├── datadog.go     # Integração com Datadog
│   │   └── elk.go         # Integração com ELK Stack
│   ├── sanitize/          # Sistema de sanitização LGPD
│   │   └── lgpd.go        # Sanitizador LGPD
│   ├── config.go          # Configuração do sistema
│   ├── logger.go          # Interface Logger principal
│   └── doc.go             # Documentação do pacote
├── examples/              # Exemplos de uso
│   ├── basic_usage.go
│   ├── middleware_example.go
│   └── observability_example.go
├── scripts/               # Scripts de configuração
├── tasks/                 # Documentação de tarefas
├── go.mod
└── README.md
```

## Dependências

### Principais
- `github.com/rs/zerolog` - Biblioteca de logging principal
- `github.com/google/uuid` - Geração de UUIDs para correlation IDs

### Middlewares HTTP
- `github.com/gin-gonic/gin` - Framework web Gin
- `github.com/gofiber/fiber/v2` - Framework web Fiber
- `github.com/go-chi/chi/v5` - Framework web Chi

### Integração PostgreSQL
- `github.com/jackc/pgx/v5` - Driver PostgreSQL

### Observabilidade
- `github.com/DataDog/datadog-go/v5/statsd` - Cliente de métricas Datadog
- `gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer` - Distributed tracing Datadog

### Rotação de Logs
- `gopkg.in/natefinch/lumberjack.v2` - Rotação automática de arquivos de log

## Contribuição

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## Licença

Este projeto está licenciado sob a licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes. 
