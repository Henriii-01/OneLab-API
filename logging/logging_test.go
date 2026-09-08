package logging

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"unknown": slog.LevelInfo,
	}

	for input, expected := range tests {
		if actual := parseLevel(input); actual != expected {
			t.Fatalf("parseLevel(%q) = %v, want %v", input, actual, expected)
		}
	}
}

func TestAccessLogRecordsResponseStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	handler := AccessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
