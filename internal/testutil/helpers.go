// Copyright © 2025 Attestant Limited.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"fmt"
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
		if _, err := io.WriteString(w, body); err != nil {
			panic(fmt.Sprintf("test handler write failed: %v", err))
		}
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
			if _, err := io.WriteString(w, endpoint.Body); err != nil {
				panic(fmt.Sprintf("test handler write failed: %v", err))
			}
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

	// Best effort close - pipe close errors are not critical in tests
	_ = w.Close()
	os.Stdout = stdout

	out, _ := io.ReadAll(r)
	return string(out)
}
