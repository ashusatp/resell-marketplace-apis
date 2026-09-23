package middlewares

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "requestID"
const requestIDHeader = "X-Request-Id"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(requestIDHeader)
		if rid == "" {
			rid = generateRequestID()
		}
		w.Header().Set(requestIDHeader, rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		ctxAddedOnRequest := r.WithContext(ctx)
		next.ServeHTTP(w, ctxAddedOnRequest)
	})
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func generateRequestID() string {
	return uuid.New().String()
}
