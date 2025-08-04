package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// HTTPTestServer creates a test HTTP server with custom handler
func HTTPTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// MockHTTPResponse creates a mock HTTP handler that returns the given response
func MockHTTPResponse(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		io.WriteString(w, body)
	}
}

// MockHTTPEndpoints creates a mock HTTP handler with different responses for different paths
func MockHTTPEndpoints(endpoints map[string]struct {
	Status int
	Body   string
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if endpoint, ok := endpoints[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(endpoint.Status)
			io.WriteString(w, endpoint.Body)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("expected string to contain %q, got %q", substr, str)
	}
}

// CaptureOutput captures stdout during test execution
func CaptureOutput(t *testing.T, f func()) string {
	t.Helper()
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = stdout

	out, _ := io.ReadAll(r)
	return string(out)
}
