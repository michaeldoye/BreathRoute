package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// Recovery returns a middleware that recovers from panics and returns a 500 error.
func Recovery(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())

					log.Error().
						Str("request_id", requestID).
						Interface("error", err).
						Str("stack", string(debug.Stack())).
						Msg("panic recovered")

					problem := models.NewInternalError(requestID, "an unexpected error occurred")
					problem.Instance = r.URL.Path
					problem.Write(w)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
