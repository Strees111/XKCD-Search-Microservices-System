package middleware

import (
	"net/http"
	"strings"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		prefix := "Token "
		if !strings.HasPrefix(header, prefix) {
			http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(header, prefix)
		if err := verifier.Verify(token); err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}
