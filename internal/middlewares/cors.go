package middlewares

import (
	"net/http"
	"strings"
)

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		clean := strings.TrimRight(strings.TrimSpace(o), "/")
		if clean != "" {
			allowedMap[clean] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			cleanOrigin := strings.TrimRight(origin, "/")

			// Check if incoming request Origin matches any allowed origin
			if origin != "" && allowedMap[cleanOrigin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				if origin != "" && !allowedMap[cleanOrigin] {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
