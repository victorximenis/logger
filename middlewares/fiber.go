package middlewares

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/victorximenis/logger/core"
)

// FiberMiddleware cria um middleware do Fiber para logging de requisições HTTP
func FiberMiddleware(config MiddlewareConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Verificar se o path deve ser ignorado
		if shouldSkipPath(c.Path(), config.SkipPaths) {
			return c.Next()
		}

		// Verificar sampling rate
		if !shouldSample(config.SamplingRate) {
			return c.Next()
		}

		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Extrair ou gerar request ID
		requestID := extractOrGenerateRequestIDFiber(c)

		// Criar contexto com request ID
		ctx := core.WithCorrelationID(c.Context(), requestID)
		c.SetUserContext(ctx)

		// Log da requisição
		logRequestFiber(ctx, config, c, requestID, method, path)

		// Processar requisição
		err := c.Next()

		// Calcular duração
		duration := time.Since(start)

		// Log da resposta
		logResponseFiber(ctx, config, c, requestID, method, path, duration)

		return err
	}
}

// extractOrGenerateRequestIDFiber extrai request ID existente ou gera um novo para Fiber
func extractOrGenerateRequestIDFiber(c *fiber.Ctx) string {
	// Criar wrapper para compatibilidade com a função utilitária
	getHeader := func(header string) string {
		return c.Get(header)
	}

	// Usar função utilitária para extrair request ID
	requestID := extractRequestIDFromHeaders(getHeader)

	// Gerar novo se não encontrado
	if requestID == "" {
		requestID = GenerateRequestID()
	}

	// Adicionar ao header de resposta
	c.Set("X-Request-ID", requestID)

	return requestID
}

// logRequestFiber faz o log da requisição HTTP para Fiber
func logRequestFiber(ctx context.Context, config MiddlewareConfig, c *fiber.Ctx, requestID, method, path string) {
	fields := map[string]interface{}{
		"component":  "http_middleware",
		"type":       "request",
		"method":     method,
		"path":       path,
		"request_id": requestID,
		"user_agent": c.Get("User-Agent"),
		"remote_ip":  c.IP(),
	}

	// Adicionar query parameters se existirem
	if len(c.Request().URI().QueryString()) > 0 {
		fields["query"] = string(c.Request().URI().QueryString())
	}

	// Adicionar headers configurados
	addHeadersFiber(fields, c, config)

	// Adicionar body se habilitado
	if config.LogRequestBody && len(c.Body()) > 0 && int64(len(c.Body())) <= config.MaxBodySize {
		addRequestBodyFiber(fields, c, config)
	}

	config.Logger.Log(ctx, core.INFO, "HTTP request started", fields)
}

// logResponseFiber faz o log da resposta HTTP para Fiber
func logResponseFiber(ctx context.Context, config MiddlewareConfig, c *fiber.Ctx, requestID, method, path string, duration time.Duration) {
	fields := map[string]interface{}{
		"component":   "http_middleware",
		"type":        "response",
		"method":      method,
		"path":        path,
		"request_id":  requestID,
		"status":      c.Response().StatusCode(),
		"size":        len(c.Response().Body()),
		"duration_ms": duration.Milliseconds(),
	}

	// Adicionar body da resposta se habilitado
	if config.LogResponseBody && len(c.Response().Body()) > 0 {
		addResponseBodyFiber(fields, c, config)
	}

	// Determinar nível do log baseado no status
	level := core.INFO
	status := c.Response().StatusCode()
	if status >= 400 && status < 500 {
		level = core.WARN
	} else if status >= 500 {
		level = core.ERROR
	}

	config.Logger.Log(ctx, level, "HTTP request completed", fields)
}

// addHeadersFiber adiciona headers configurados aos campos do log para Fiber
func addHeadersFiber(fields map[string]interface{}, c *fiber.Ctx, config MiddlewareConfig) {
	for _, header := range config.LoggedHeaders {
		value := c.Get(header)
		if value != "" {
			key := normalizeHeaderName(header)
			value = sanitizeHeaderValue(header, value, config)
			fields[key] = value
		}
	}
}

// addRequestBodyFiber adiciona o body da requisição aos campos do log para Fiber
func addRequestBodyFiber(fields map[string]interface{}, c *fiber.Ctx, config MiddlewareConfig) {
	body := c.Body()
	if len(body) == 0 {
		return
	}

	// Sanitizar o body
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["request_body"] = string(sanitizedBody)
}

// addResponseBodyFiber adiciona o body da resposta aos campos do log para Fiber
func addResponseBodyFiber(fields map[string]interface{}, c *fiber.Ctx, config MiddlewareConfig) {
	body := c.Response().Body()
	if len(body) == 0 {
		return
	}

	// Sanitizar o body
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["response_body"] = string(sanitizedBody)
}
