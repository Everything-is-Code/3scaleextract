package config

import (
	"crypto/tls"
	"net/http"
	"time"
)

// NewHTTPClient returns an HTTP client for Admin API calls.
func NewHTTPClient(insecureTLS bool, timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if insecureTLS {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}
	return client
}
