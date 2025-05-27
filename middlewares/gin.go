package middlewares

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/victorximenis/logger/core"
)

// MiddlewareConfig define a configuração para o middleware do Gin
type MiddlewareConfig struct {
	// LoggedHeaders define quais headers devem ser logados
	LoggedHeaders []string
	// SensitiveHeaders define headers que devem ser mascarados
	SensitiveHeaders []string
	// SensitiveFields define campos que devem ser sanitizados no body
	SensitiveFields []string
	// LogRequestBody habilita logging do body da requisição
	LogRequestBody bool
	// LogResponseBody habilita logging do body da resposta
	LogResponseBody bool
	// MaxBodySize define o tamanho máximo do body para logging (em bytes)
	MaxBodySize int64
	// SamplingRate define a taxa de amostragem para logs (0.0 a 1.0)
	SamplingRate float64
	// Logger define o logger a ser usado
	Logger core.LoggerAdapter
	// SkipPaths define paths que devem ser ignorados pelo middleware
	SkipPaths []string
	// SensitiveHeaderPatterns define padrões regex para headers sensíveis
	SensitiveHeaderPatterns []*regexp.Regexp
}

// DefaultMiddlewareConfig retorna uma configuração padrão para o middleware
func DefaultMiddlewareConfig(logger core.LoggerAdapter) MiddlewareConfig {
	return MiddlewareConfig{
		LoggedHeaders: []string{
			"User-Agent", "Content-Type", "Accept", "Accept-Language",
			"X-Forwarded-For", "X-Real-IP", "X-Request-ID",
		},
		SensitiveHeaders: []string{
			"Authorization", "Cookie", "Set-Cookie", "X-API-Key",
			"X-Auth-Token", "Bearer", "Basic",
		},
		SensitiveFields: []string{
			"password", "senha", "token", "secret", "api_key",
			"credit_card", "cpf", "cnpj", "authorization",
		},
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodySize:     1024 * 1024, // 1MB
		SamplingRate:    1.0,         // 100% por padrão
		Logger:          logger,
		SkipPaths: []string{
			"/health", "/metrics", "/ping", "/favicon.ico",
		},
		SensitiveHeaderPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)authorization`),
			regexp.MustCompile(`(?i)cookie`),
			regexp.MustCompile(`(?i)token`),
			regexp.MustCompile(`(?i)api[_-]?key`),
			regexp.MustCompile(`(?i)secret`),
		},
	}
}

// WithLoggedHeaders configura os headers que devem ser logados
func (c MiddlewareConfig) WithLoggedHeaders(headers ...string) MiddlewareConfig {
	c.LoggedHeaders = headers
	return c
}

// WithSensitiveHeaders configura os headers que devem ser mascarados
func (c MiddlewareConfig) WithSensitiveHeaders(headers ...string) MiddlewareConfig {
	c.SensitiveHeaders = headers
	return c
}

// WithSensitiveFields configura os campos que devem ser sanitizados
func (c MiddlewareConfig) WithSensitiveFields(fields ...string) MiddlewareConfig {
	c.SensitiveFields = fields
	return c
}

// WithRequestBodyLogging habilita/desabilita logging do body da requisição
func (c MiddlewareConfig) WithRequestBodyLogging(enabled bool) MiddlewareConfig {
	c.LogRequestBody = enabled
	return c
}

// WithResponseBodyLogging habilita/desabilita logging do body da resposta
func (c MiddlewareConfig) WithResponseBodyLogging(enabled bool) MiddlewareConfig {
	c.LogResponseBody = enabled
	return c
}

// WithMaxBodySize configura o tamanho máximo do body para logging
func (c MiddlewareConfig) WithMaxBodySize(size int64) MiddlewareConfig {
	c.MaxBodySize = size
	return c
}

// WithSamplingRate configura a taxa de amostragem
func (c MiddlewareConfig) WithSamplingRate(rate float64) MiddlewareConfig {
	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}
	c.SamplingRate = rate
	return c
}

// WithSkipPaths configura paths que devem ser ignorados
func (c MiddlewareConfig) WithSkipPaths(paths ...string) MiddlewareConfig {
	c.SkipPaths = paths
	return c
}

// GinMiddleware cria um middleware do Gin para logging de requisições HTTP
func GinMiddleware(config MiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se o path deve ser ignorado
		if shouldSkipPath(c.Request.URL.Path, config.SkipPaths) {
			c.Next()
			return
		}

		// Verificar sampling rate
		if !shouldSample(config.SamplingRate) {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Extrair ou gerar request ID
		requestID := extractOrGenerateRequestID(c)

		// Criar contexto com request ID
		ctx := core.WithCorrelationID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log da requisição
		logRequest(ctx, config, c, requestID, method, path)

		// Criar response writer que captura a resposta
		responseWriter := &responseLogWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			maxSize:        config.MaxBodySize,
		}
		c.Writer = responseWriter

		// Processar requisição
		c.Next()

		// Calcular duração
		duration := time.Since(start)

		// Log da resposta
		logResponse(ctx, config, responseWriter, requestID, method, path, duration)
	}
}

// extractOrGenerateRequestID extrai request ID existente ou gera um novo
func extractOrGenerateRequestID(c *gin.Context) string {
	// Usar função utilitária para extrair request ID
	requestID := extractRequestIDFromHeaders(c.GetHeader)

	// Gerar novo se não encontrado
	if requestID == "" {
		requestID = GenerateRequestID()
	}

	// Adicionar ao header de resposta
	c.Header("X-Request-ID", requestID)

	return requestID
}

// logRequest faz o log da requisição HTTP
func logRequest(ctx context.Context, config MiddlewareConfig, c *gin.Context, requestID, method, path string) {
	fields := map[string]interface{}{
		"component":  "http_middleware",
		"type":       "request",
		"method":     method,
		"path":       path,
		"request_id": requestID,
		"user_agent": c.GetHeader("User-Agent"),
		"remote_ip":  c.ClientIP(),
	}

	// Adicionar query parameters se existirem
	if len(c.Request.URL.RawQuery) > 0 {
		fields["query"] = c.Request.URL.RawQuery
	}

	// Adicionar headers configurados
	addHeaders(fields, c, config)

	// Adicionar body se habilitado
	if config.LogRequestBody && c.Request.ContentLength > 0 && c.Request.ContentLength <= config.MaxBodySize {
		addRequestBody(fields, c, config)
	}

	config.Logger.Log(ctx, core.INFO, "HTTP request started", fields)
}

// logResponse faz o log da resposta HTTP
func logResponse(ctx context.Context, config MiddlewareConfig, writer *responseLogWriter, requestID, method, path string, duration time.Duration) {
	fields := map[string]interface{}{
		"component":   "http_middleware",
		"type":        "response",
		"method":      method,
		"path":        path,
		"request_id":  requestID,
		"status":      writer.Status(),
		"size":        writer.Size(),
		"duration_ms": duration.Milliseconds(),
	}

	// Adicionar body da resposta se habilitado
	if config.LogResponseBody && writer.body.Len() > 0 {
		addResponseBody(fields, writer, config)
	}

	// Determinar nível do log baseado no status
	level := core.INFO
	if writer.Status() >= 400 && writer.Status() < 500 {
		level = core.WARN
	} else if writer.Status() >= 500 {
		level = core.ERROR
	}

	config.Logger.Log(ctx, level, "HTTP request completed", fields)
}

// addHeaders adiciona headers configurados aos campos do log
func addHeaders(fields map[string]interface{}, c *gin.Context, config MiddlewareConfig) {
	for _, header := range config.LoggedHeaders {
		value := c.GetHeader(header)
		if value != "" {
			key := normalizeHeaderName(header)
			value = sanitizeHeaderValue(header, value, config)
			fields[key] = value
		}
	}
}

// addRequestBody adiciona o body da requisição aos campos do log
func addRequestBody(fields map[string]interface{}, c *gin.Context, config MiddlewareConfig) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		fields["request_body_error"] = err.Error()
		return
	}

	// Restaurar o body para os handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Sanitizar o body
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["request_body"] = string(sanitizedBody)
}

// addResponseBody adiciona o body da resposta aos campos do log
func addResponseBody(fields map[string]interface{}, writer *responseLogWriter, config MiddlewareConfig) {
	body := writer.body.Bytes()
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["response_body"] = string(sanitizedBody)
}

// responseLogWriter é um wrapper do ResponseWriter que captura a resposta
type responseLogWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int64
	size    int
}

// Write implementa io.Writer
func (w *responseLogWriter) Write(data []byte) (int, error) {
	// Capturar body se não exceder o tamanho máximo
	if w.body.Len()+len(data) <= int(w.maxSize) {
		w.body.Write(data)
	}

	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}

// WriteString implementa io.StringWriter
func (w *responseLogWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Status retorna o status code da resposta
func (w *responseLogWriter) Status() int {
	return w.ResponseWriter.Status()
}

// Size retorna o tamanho da resposta em bytes
func (w *responseLogWriter) Size() int {
	return w.size
}
