package wacloudapi

import (
	"net/http"
	"time"
)

type Option func(*Config)

func WithAPIVersion(version string) Option {
	return func(c *Config) {
		if version != "" {
			c.APIVersion = version
		}
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Config) {
		if baseURL != "" {
			c.BaseURL = baseURL
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		if client != nil {
			c.HTTPClient = client
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		if c.HTTPClient == nil {
			c.HTTPClient = &http.Client{}
		}
		c.HTTPClient.Timeout = timeout
	}
}

func WithRetry(maxRetries int, waitMin ...time.Duration) Option {
	return func(c *Config) {
		if maxRetries > 0 {
			c.MaxRetries = maxRetries
		}
		if len(waitMin) > 0 && waitMin[0] > 0 {
			c.RetryWaitMin = waitMin[0]
		}
	}
}
