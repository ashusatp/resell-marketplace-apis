package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/tecisage/resell-apis/internal/httpx"
	"github.com/tecisage/resell-apis/internal/tokens"
)

type authContextKey struct {
	UserID string
	Email  string
}

var (
	contextKeyClaims = authContextKey{}
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpx.Error(w, http.StatusUnauthorized, "Missing Authorization header", httpx.CodeUnauthorized)
				return
			}

			// Expected format: Bearer <token>
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				httpx.Error(w, http.StatusUnauthorized, "Invalid Authorization header", httpx.CodeUnauthorized)
				return
			}

			token := parts[1]
			claims, err := tokens.Validate(token, jwtSecret)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "Unauthorized", httpx.CodeUnauthorized)
				return
			}

			newCtx := context.WithValue(r.Context(), contextKeyClaims, authContextKey{
				UserID: claims.UserID,
				Email:  claims.Email,
			})

			r = r.WithContext(newCtx)

			next.ServeHTTP(w, r)
		})
	}
}

func GetAuthClaims(ctx context.Context) (*authContextKey, error) {
	val := ctx.Value(contextKeyClaims)
	if val == nil {
		return nil, errors.New("no claims found in context")
	}
	claims, ok := val.(authContextKey)
	if !ok {
		return nil, errors.New("invalid claims type")
	}
	return &claims, nil
}
