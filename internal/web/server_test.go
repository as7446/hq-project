package web

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 1
func TestIndexShowsBuildVersion(t *testing.T) {
	cfg := Config{
		ServiceName: "demo",
		Port:        "8080",
		JobName:     "hq-project/main",
		BuildNumber: "42",
		GitCommit:   "abcdef1234567890",
		Environment: "test",
	}
	handler := NewServer(cfg, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"demo", "hq-project/main#42@abcdef123456", "test"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response to contain %q, got %s", want, body)
		}
	}
}

func TestLogSubmitRequiresMessage(t *testing.T) {
	handler := NewServer(Config{ServiceName: "demo"}, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/logs", strings.NewReader("message="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	handler := NewServer(Config{ServiceName: "demo"}, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
