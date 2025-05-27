package middlewares

import (
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/victorximenis/logger/sanitize"
)

var (
	// Cache de padrões regex compilados para performance
	regexCache = make(map[string]*regexp.Regexp)
	regexMutex sync.RWMutex
)

// GenerateRequestID gera um novo ID único para requisições
func GenerateRequestID() string {
	return uuid.New().String()
}

// shouldSkipPath verifica se um path deve ser ignorado
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// shouldSample verifica se deve fazer sampling baseado na taxa configurada
func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}
	// Implementação simples de sampling baseada no UUID
	id := uuid.New()
	hash := float64(id[0]) / 255.0
	return hash < rate
}

// isSensitiveHeader verifica se um header é sensível
func isSensitiveHeader(header string, config MiddlewareConfig) bool {
	headerLower := strings.ToLower(header)

	// Verificar lista de headers sensíveis
	for _, sensitive := range config.SensitiveHeaders {
		if strings.ToLower(sensitive) == headerLower {
			return true
		}
	}

	// Verificar padrões regex
	for _, pattern := range config.SensitiveHeaderPatterns {
		if pattern.MatchString(header) {
			return true
		}
	}

	return false
}

// maskSensitiveData mascara dados sensíveis
func maskSensitiveData(data string) string {
	if len(data) <= 8 {
		return "***"
	}
	return data[:4] + "***"
}

// sanitizeBody sanitiza o body usando o sistema de sanitização
func sanitizeBody(body []byte, sensitiveFields []string) []byte {
	if len(body) == 0 {
		return body
	}

	// Tentar sanitizar como JSON
	config := sanitize.DefaultSensitiveFieldConfig()

	// Adicionar campos sensíveis customizados
	config.MaskCompletely = append(config.MaskCompletely, sensitiveFields...)

	if sanitized, err := sanitize.SanitizeJSON(body, config); err == nil {
		return sanitized
	}

	// Se não for JSON válido, sanitizar como string
	sanitizedString := sanitize.SanitizeString(string(body), config)
	return []byte(sanitizedString)
}

// normalizeHeaderName normaliza o nome de um header para uso em logs
func normalizeHeaderName(header string) string {
	return "header_" + strings.ToLower(strings.ReplaceAll(header, "-", "_"))
}

// isJSONContent verifica se o content type é JSON
func isJSONContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "application/json")
}

// isXMLContent verifica se o content type é XML
func isXMLContent(contentType string) bool {
	contentTypeLower := strings.ToLower(contentType)
	return strings.Contains(contentTypeLower, "application/xml") ||
		strings.Contains(contentTypeLower, "text/xml")
}

// isTextContent verifica se o content type é texto
func isTextContent(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "text/")
}

// shouldLogBody verifica se o body deve ser logado baseado no content type e tamanho
func shouldLogBody(contentType string, bodySize int64, maxSize int64) bool {
	if bodySize <= 0 || bodySize > maxSize {
		return false
	}

	// Logar apenas content types conhecidos e seguros
	return isJSONContent(contentType) || isXMLContent(contentType) || isTextContent(contentType)
}

// getCompiledRegex retorna um regex compilado do cache ou compila e armazena
func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	regexMutex.RLock()
	if regex, exists := regexCache[pattern]; exists {
		regexMutex.RUnlock()
		return regex, nil
	}
	regexMutex.RUnlock()

	// Compilar regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	// Armazenar no cache
	regexMutex.Lock()
	regexCache[pattern] = regex
	regexMutex.Unlock()

	return regex, nil
}

// extractRequestIDFromHeaders extrai request ID de headers comuns
func extractRequestIDFromHeaders(getHeader func(string) string) string {
	// Tentar extrair de headers comuns em ordem de prioridade
	headers := []string{"X-Request-ID", "X-Correlation-ID", "X-Trace-ID"}

	for _, header := range headers {
		if requestID := getHeader(header); requestID != "" {
			return requestID
		}
	}

	return ""
}

// sanitizeHeaderValue sanitiza o valor de um header se necessário
func sanitizeHeaderValue(header, value string, config MiddlewareConfig) string {
	if isSensitiveHeader(header, config) {
		return maskSensitiveData(value)
	}
	return value
}

// buildLogFields cria um mapa base de campos para logging
func buildLogFields(component, logType, method, path, requestID string) map[string]interface{} {
	return map[string]interface{}{
		"component":  component,
		"type":       logType,
		"method":     method,
		"path":       path,
		"request_id": requestID,
	}
}

// addQueryParams adiciona query parameters aos campos de log se existirem
func addQueryParams(fields map[string]interface{}, query string) {
	if len(query) > 0 {
		fields["query"] = query
	}
}

// truncateString trunca uma string se exceder o tamanho máximo
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// sanitizeLogValue sanitiza um valor antes de incluí-lo nos logs
func sanitizeLogValue(key string, value interface{}, sensitiveFields []string) interface{} {
	// Verificar se a chave é sensível
	keyLower := strings.ToLower(key)
	for _, field := range sensitiveFields {
		if strings.Contains(keyLower, strings.ToLower(field)) {
			if str, ok := value.(string); ok {
				return maskSensitiveData(str)
			}
			return "***"
		}
	}
	return value
}
