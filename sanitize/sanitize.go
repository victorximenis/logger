package sanitize

import (
	"encoding/json"
	"regexp"
	"strings"
)

// SensitiveFieldConfig define como tratar campos sensíveis
type SensitiveFieldConfig struct {
	// Campos para mascarar completamente (substituir por "***")
	MaskCompletely []string

	// Campos para mascarar parcialmente (mostrar primeiros/últimos caracteres)
	MaskPartially []string

	// Expressões regulares para identificar padrões sensíveis
	Patterns map[string]*regexp.Regexp
}

// DefaultSensitiveFieldConfig retorna a configuração padrão para campos sensíveis
func DefaultSensitiveFieldConfig() SensitiveFieldConfig {
	return SensitiveFieldConfig{
		MaskCompletely: []string{
			"password", "senha", "secret", "token", "api_key", "apikey",
			"credit_card", "cartao", "cvv", "authorization", "bearer",
		},
		MaskPartially: []string{
			"cpf", "cnpj", "email", "phone", "telefone", "celular",
			"address", "endereco", "cep", "zipcode", "rg", "documento",
		},
		Patterns: map[string]*regexp.Regexp{
			"cpf":   regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`),
			"cnpj":  regexp.MustCompile(`\d{2}\.?\d{3}\.?\d{3}/?0001-?\d{2}`),
			"email": regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			"card":  regexp.MustCompile(`\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`),
			"phone": regexp.MustCompile(`\(?(\d{2})\)?\s?9?\d{4}-?\d{4}`),
		},
	}
}

// SanitizeJSON sanitiza dados sensíveis em uma string JSON
func SanitizeJSON(jsonData []byte, config SensitiveFieldConfig) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}

	sanitized := sanitizeValue(data, "", config)
	return json.Marshal(sanitized)
}

// SanitizeString sanitiza uma string individual usando as regras de configuração
func SanitizeString(data string, config SensitiveFieldConfig) string {
	return sanitizeString(data, "", config)
}

// sanitizeValue sanitiza recursivamente valores em uma estrutura de dados
func sanitizeValue(data interface{}, path string, config SensitiveFieldConfig) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return sanitizeMap(v, path, config)
	case []interface{}:
		return sanitizeArray(v, path, config)
	case string:
		return sanitizeString(v, path, config)
	default:
		return v
	}
}

// sanitizeMap sanitiza um mapa
func sanitizeMap(data map[string]interface{}, path string, config SensitiveFieldConfig) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		fieldPath := path
		if fieldPath != "" {
			fieldPath += "."
		}
		fieldPath += strings.ToLower(k)

		// Verificar se este campo deve ser completamente mascarado
		if shouldMaskCompletely(fieldPath, config) {
			result[k] = "***"
			continue
		}

		// Verificar se este campo deve ser parcialmente mascarado
		if shouldMaskPartially(fieldPath, config) {
			if str, ok := v.(string); ok {
				result[k] = maskPartially(str)
				continue
			}
		}

		// Sanitizar recursivamente o valor
		result[k] = sanitizeValue(v, fieldPath, config)
	}

	return result
}

// sanitizeArray sanitiza um array
func sanitizeArray(data []interface{}, path string, config SensitiveFieldConfig) []interface{} {
	result := make([]interface{}, len(data))

	for i, v := range data {
		result[i] = sanitizeValue(v, path, config)
	}

	return result
}

// sanitizeString sanitiza um valor string
func sanitizeString(data string, path string, config SensitiveFieldConfig) string {
	// Verificar se este campo deve ser completamente mascarado
	if shouldMaskCompletely(path, config) {
		return "***"
	}

	// Verificar se este campo deve ser parcialmente mascarado
	if shouldMaskPartially(path, config) {
		return maskPartially(data)
	}

	// Verificar padrões sensíveis na string
	for _, pattern := range config.Patterns {
		if pattern.MatchString(data) {
			return maskSensitivePattern(data, pattern)
		}
	}

	return data
}

// shouldMaskCompletely verifica se um campo deve ser completamente mascarado
func shouldMaskCompletely(path string, config SensitiveFieldConfig) bool {
	for _, field := range config.MaskCompletely {
		if strings.Contains(path, field) {
			return true
		}
	}
	return false
}

// shouldMaskPartially verifica se um campo deve ser parcialmente mascarado
func shouldMaskPartially(path string, config SensitiveFieldConfig) bool {
	for _, field := range config.MaskPartially {
		if strings.Contains(path, field) {
			return true
		}
	}
	return false
}

// maskPartially mascara parte de uma string
func maskPartially(data string) string {
	if len(data) <= 4 {
		return "***"
	}

	// Mostrar primeiros 2 e últimos 2 caracteres
	return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}

// maskSensitivePattern mascara padrões sensíveis como CPF, CNPJ, etc.
func maskSensitivePattern(data string, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(data, func(match string) string {
		if len(match) <= 4 {
			return "***"
		}

		// Mostrar primeiros 2 e últimos 2 caracteres
		return match[:2] + strings.Repeat("*", len(match)-4) + match[len(match)-2:]
	})
}
