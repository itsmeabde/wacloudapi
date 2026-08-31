package wacloudapi

import (
	"net/http"
	"time"
)

const (
	DefaultAPIVersion = "v21.0"
	DefaultBaseURL    = "https://graph.facebook.com"
	DefaultTimeout    = 30 * time.Second
	DefaultWaitMin    = 500 * time.Millisecond
	DefaultWaitMax    = 5 * time.Second
)

type Config struct {
	AccessToken   string
	PhoneNumberID string
	APIVersion    string
	BaseURL       string
	HTTPClient    *http.Client
	MaxRetries    int
	RetryWaitMin  time.Duration
	RetryWaitMax  time.Duration
}

func defaultConfig(accessToken, phoneNumberID string) *Config {
	return &Config{
		AccessToken:   accessToken,
		PhoneNumberID: phoneNumberID,
		APIVersion:    DefaultAPIVersion,
		BaseURL:       DefaultBaseURL,
		HTTPClient:    &http.Client{Timeout: DefaultTimeout},
		MaxRetries:    0,
		RetryWaitMin:  DefaultWaitMin,
		RetryWaitMax:  DefaultWaitMax,
	}
}
