package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
)

// Middleware lets a request through only with a valid bearer token, putting
// the caller's Identity into its context; anything else gets a 401.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	log := logger.Named("auth")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok {
			unauthorized(w, "Authorization token is required")
			return
		}

		identity, err := v.Verify(raw)
		if err != nil {
			log.InfoContext(r.Context(), "token rejected",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			unauthorized(w, "Invalid authorization token")
			return
		}

		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{Message: message})
}
