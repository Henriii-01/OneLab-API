package logging

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const serviceName = "onelab-api"

// New creates the process logger. JSON stdout is intentional for container log collectors.
func New() *slog.Logger {
	level := parseLevel(os.Getenv("ONELAB_LOG_LEVEL"))
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", serviceName)
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// AccessLog records one structured event per HTTP request.
func AccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)

		level := slog.LevelInfo
		if writer.status >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if writer.status >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		logger.Log(r.Context(), level, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", writer.status,
			"duration_ms", time.Since(started).Seconds()*1000,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
