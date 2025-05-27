package middlewares

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/victorximenis/logger/core"
)

// ChiMiddleware cria um middleware do Chi para logging de requisições HTTP
func ChiMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verificar se o path deve ser ignorado
			if shouldSkipPath(r.URL.Path, config.SkipPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Verificar sampling rate
			if !shouldSample(config.SamplingRate) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			path := r.URL.Path
			method := r.Method

			// Extrair ou gerar request ID
			requestID := extractOrGenerateRequestIDChi(w, r)

			// Criar contexto com request ID
			ctx := core.WithCorrelationID(r.Context(), requestID)
			r = r.WithContext(ctx)

			// Log da requisição
			logRequestChi(ctx, config, r, requestID, method, path)

			// Criar response writer que captura a resposta
			responseWriter := &chiResponseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				maxSize:        config.MaxBodySize,
				status:         http.StatusOK,
			}

			// Processar requisição
			next.ServeHTTP(responseWriter, r)

			// Calcular duração
			duration := time.Since(start)

			// Log da resposta
			logResponseChi(ctx, config, responseWriter, requestID, method, path, duration)
		})
	}
}

// extractOrGenerateRequestIDChi extrai request ID existente ou gera um novo para Chi
func extractOrGenerateRequestIDChi(w http.ResponseWriter, r *http.Request) string {
	// Usar função utilitária para extrair request ID
	requestID := extractRequestIDFromHeaders(r.Header.Get)

	// Gerar novo se não encontrado
	if requestID == "" {
		requestID = GenerateRequestID()
	}

	// Adicionar ao header de resposta
	w.Header().Set("X-Request-ID", requestID)

	return requestID
}

// logRequestChi faz o log da requisição HTTP para Chi
func logRequestChi(ctx context.Context, config MiddlewareConfig, r *http.Request, requestID, method, path string) {
	fields := map[string]interface{}{
		"component":  "http_middleware",
		"type":       "request",
		"method":     method,
		"path":       path,
		"request_id": requestID,
		"user_agent": r.Header.Get("User-Agent"),
		"remote_ip":  getClientIPChi(r),
	}

	// Adicionar query parameters se existirem
	if len(r.URL.RawQuery) > 0 {
		fields["query"] = r.URL.RawQuery
	}

	// Adicionar headers configurados
	addHeadersChi(fields, r, config)

	// Adicionar body se habilitado
	if config.LogRequestBody && r.ContentLength > 0 && r.ContentLength <= config.MaxBodySize {
		addRequestBodyChi(fields, r, config)
	}

	config.Logger.Log(ctx, core.INFO, "HTTP request started", fields)
}

// logResponseChi faz o log da resposta HTTP para Chi
func logResponseChi(ctx context.Context, config MiddlewareConfig, writer *chiResponseWriter, requestID, method, path string, duration time.Duration) {
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
		addResponseBodyChi(fields, writer, config)
	}

	// Determinar nível do log baseado no status
	level := core.INFO
	status := writer.Status()
	if status >= 400 && status < 500 {
		level = core.WARN
	} else if status >= 500 {
		level = core.ERROR
	}

	config.Logger.Log(ctx, level, "HTTP request completed", fields)
}

// addHeadersChi adiciona headers configurados aos campos do log para Chi
func addHeadersChi(fields map[string]interface{}, r *http.Request, config MiddlewareConfig) {
	for _, header := range config.LoggedHeaders {
		value := r.Header.Get(header)
		if value != "" {
			key := normalizeHeaderName(header)
			value = sanitizeHeaderValue(header, value, config)
			fields[key] = value
		}
	}
}

// addRequestBodyChi adiciona o body da requisição aos campos do log para Chi
func addRequestBodyChi(fields map[string]interface{}, r *http.Request, config MiddlewareConfig) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fields["request_body_error"] = err.Error()
		return
	}

	// Restaurar o body para os handlers
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Sanitizar o body
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["request_body"] = string(sanitizedBody)
}

// addResponseBodyChi adiciona o body da resposta aos campos do log para Chi
func addResponseBodyChi(fields map[string]interface{}, writer *chiResponseWriter, config MiddlewareConfig) {
	body := writer.body.Bytes()
	sanitizedBody := sanitizeBody(body, config.SensitiveFields)
	fields["response_body"] = string(sanitizedBody)
}

// getClientIPChi extrai o IP do cliente para Chi
func getClientIPChi(r *http.Request) string {
	// Verificar headers de proxy
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For pode conter múltiplos IPs, pegar o primeiro
		if idx := strings.Index(ip, ","); idx != -1 {
			return strings.TrimSpace(ip[:idx])
		}
		return strings.TrimSpace(ip)
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Fallback para RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// chiResponseWriter é um wrapper do ResponseWriter que captura a resposta para Chi
type chiResponseWriter struct {
	http.ResponseWriter
	body    *bytes.Buffer
	maxSize int64
	size    int
	status  int
}

// Write implementa io.Writer
func (w *chiResponseWriter) Write(data []byte) (int, error) {
	// Capturar body se não exceder o tamanho máximo
	if w.body.Len()+len(data) <= int(w.maxSize) {
		w.body.Write(data)
	}

	n, err := w.ResponseWriter.Write(data)
	w.size += n
	return n, err
}

// WriteHeader implementa http.ResponseWriter
func (w *chiResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Status retorna o status code da resposta
func (w *chiResponseWriter) Status() int {
	return w.status
}

// Size retorna o tamanho da resposta em bytes
func (w *chiResponseWriter) Size() int {
	return w.size
}

// Header implementa http.ResponseWriter
func (w *chiResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Flush implementa http.Flusher se o ResponseWriter original suportar
func (w *chiResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implementa http.Hijacker se o ResponseWriter original suportar
func (w *chiResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
