package logger

import (
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func LoggerMiddleware(log *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				log.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration", time.Since(start).Milliseconds(),
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
