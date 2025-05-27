package core

import "context"

// contextKey é um tipo personalizado para chaves de contexto para evitar colisões
type contextKey string

// Constantes para chaves de contexto padrão
const (
	traceIDKey       contextKey = "trace_id"
	correlationIDKey contextKey = "correlation_id"
	userIDKey        contextKey = "user_id"
)

// WithTraceID adiciona um trace ID ao contexto
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithCorrelationID adiciona um correlation ID ao contexto
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// WithUserID adiciona um user ID ao contexto
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetTraceID extrai o trace ID do contexto
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok && traceID != ""
}

// GetCorrelationID extrai o correlation ID do contexto
func GetCorrelationID(ctx context.Context) (string, bool) {
	correlationID, ok := ctx.Value(correlationIDKey).(string)
	return correlationID, ok && correlationID != ""
}

// GetUserID extrai o user ID do contexto
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok && userID != ""
}
