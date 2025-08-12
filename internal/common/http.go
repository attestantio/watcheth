package common

import (
	"net/http"
	"time"
)

// DefaultHTTPTimeout is the default timeout for HTTP requests
const DefaultHTTPTimeout = 10 * time.Second

// NewHTTPClient creates a new HTTP client with sensible defaults
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}
