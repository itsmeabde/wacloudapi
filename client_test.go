package wacloudapi

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := New("test-token", "123456789")
	if c.config.AccessToken != "test-token" {
		t.Errorf("expected token 'test-token', got %s", c.config.AccessToken)
	}
	if c.config.PhoneNumberID != "123456789" {
		t.Errorf("expected phone number ID '123456789', got %s", c.config.PhoneNumberID)
	}
	if c.config.APIVersion != "v21.0" {
		t.Errorf("expected default version 'v21.0', got %s", c.config.APIVersion)
	}
	if c.config.BaseURL != "https://graph.facebook.com" {
		t.Errorf("expected default BaseURL, got %s", c.config.BaseURL)
	}
	if c.config.HTTPClient == nil {
		t.Errorf("expected non-nil default HTTPClient")
	}
}

func TestNewClientCustomOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 10 * time.Second}
	c := New("test-token", "123456789",
		WithAPIVersion("v22.0"),
		WithBaseURL("https://custom.graph.com"),
		WithHTTPClient(customHTTP),
		WithRetry(3, 100*time.Millisecond),
	)

	if c.config.APIVersion != "v22.0" {
		t.Errorf("expected custom APIVersion 'v22.0', got %s", c.config.APIVersion)
	}
	if c.config.BaseURL != "https://custom.graph.com" {
		t.Errorf("expected custom BaseURL, got %s", c.config.BaseURL)
	}
	if c.config.HTTPClient != customHTTP {
		t.Errorf("expected custom HTTPClient")
	}
	if c.config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", c.config.MaxRetries)
	}
	if c.config.RetryWaitMin != 100*time.Millisecond {
		t.Errorf("expected RetryWaitMin 100ms, got %v", c.config.RetryWaitMin)
	}
}

func TestWithTimeoutOption(t *testing.T) {
	c := New("test-token", "123456789", WithTimeout(15*time.Second))
	if c.config.HTTPClient.Timeout != 15*time.Second {
		t.Errorf("expected timeout 15s, got %v", c.config.HTTPClient.Timeout)
	}

	// Test WithTimeout when HTTPClient is nil
	cfg := &Config{}
	WithTimeout(5 * time.Second)(cfg)
	if cfg.HTTPClient == nil || cfg.HTTPClient.Timeout != 5*time.Second {
		t.Errorf("expected initialized HTTPClient with timeout 5s")
	}
}

func TestOptionsEmptyOrZero(t *testing.T) {
	c := New("token", "pid",
		WithAPIVersion(""),
		WithBaseURL(""),
		WithHTTPClient(nil),
		WithRetry(0),
	)

	if c.config.APIVersion != DefaultAPIVersion {
		t.Errorf("expected default APIVersion when empty passed, got %s", c.config.APIVersion)
	}
	if c.config.BaseURL != DefaultBaseURL {
		t.Errorf("expected default BaseURL when empty passed, got %s", c.config.BaseURL)
	}
	if c.config.MaxRetries != 0 {
		t.Errorf("expected MaxRetries 0, got %d", c.config.MaxRetries)
	}
}

func TestNewClient_NilOption(t *testing.T) {
	// Should not panic with nil option
	c := New("token", "pid", nil, WithAPIVersion("v22.0"), nil)
	if c.config.APIVersion != "v22.0" {
		t.Errorf("expected APIVersion v22.0, got %s", c.config.APIVersion)
	}
}
