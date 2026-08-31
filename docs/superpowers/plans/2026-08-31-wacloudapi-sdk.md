# Go WhatsApp Cloud API SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun package Go `github.com/itsmeabde/wacloudapi` yang modular, zero-dependency, type-safe, dan andal untuk berinteraksi dengan Meta WhatsApp Cloud API (Graph API v21.0+), mencakup Messaging Service, Media Management, dan Webhook Handler/Parser.

**Architecture:** Menggunakan arsitektur modular *service-based client* dengan *functional options* di root package (`wacloudapi.New()`, `client.Messages`, `client.Media`), penanganan error terstruktur (`*APIError`) dengan built-in retry backoff pada internal HTTP runner, serta sub-package `webhook` terisolasi untuk verifikasi signature HMAC-SHA256, challenge verification, dan event dispatcher bertipe kuat.

**Tech Stack:** Go 1.22+ (Zero external dependencies, 100% Go Standard Library: `net/http`, `crypto/hmac`, `crypto/sha256`, `encoding/json`, `io`, `time`, `context`, `mime/multipart`).

## Global Constraints
- Go Version: `go 1.22+`
- External Dependencies: 0 (Zero external dependencies, only Go standard library)
- Module Name: `github.com/itsmeabde/wacloudapi`
- Default API Version: `v21.0`
- Default Base URL: `https://graph.facebook.com`
- Concurrency: All exported client and handler methods must be safe for concurrent goroutine access
- Context: All API network calls must accept and respect `context.Context`

---

### Task 1: Core Config, Options, Error Types, and Client Struct

**Files:**
- Create: `config.go`
- Create: `options.go`
- Create: `error.go`
- Create: `client.go`
- Test: `error_test.go`
- Test: `client_test.go`

**Interfaces:**
- Produces:
  - `Config`, `Option`, `WithAPIVersion(string)`, `WithBaseURL(string)`, `WithHTTPClient(*http.Client)`, `WithTimeout(time.Duration)`, `WithRetry(int, ...time.Duration)`
  - `APIError`, `(e *APIError) Error() string`, `(e *APIError) IsRateLimit() bool`, `(e *APIError) IsAuthError() bool`, `(e *APIError) IsTemplateError() bool`
  - `Client`, `New(accessToken, phoneNumberID string, opts ...Option) *Client`

- [ ] **Step 1: Write failing tests for Config, Options, APIError, and Client**

Create `error_test.go`:
```go
package wacloudapi

import (
	"testing"
)

func TestAPIErrorPredicates(t *testing.T) {
	rateLimitErr := &APIError{HTTPStatusCode: 429, Code: 80007, Message: "Rate limit hit"}
	if !rateLimitErr.IsRateLimit() {
		t.Errorf("expected IsRateLimit to be true for code 80007 / 429")
	}

	authErr := &APIError{HTTPStatusCode: 401, Code: 190, Message: "Invalid OAuth access token"}
	if !authErr.IsAuthError() {
		t.Errorf("expected IsAuthError to be true for code 190")
	}

	tplErr := &APIError{HTTPStatusCode: 400, Code: 132000, Message: "Template param mismatch"}
	if !tplErr.IsTemplateError() {
		t.Errorf("expected IsTemplateError to be true for code 132000")
	}

	if rateLimitErr.Error() == "" {
		t.Errorf("expected non-empty Error() string")
	}
}
```

Create `client_test.go`:
```go
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
	if c.config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", c.config.MaxRetries)
	}
	if c.config.RetryWaitMin != 100*time.Millisecond {
		t.Errorf("expected RetryWaitMin 100ms, got %v", c.config.RetryWaitMin)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./...`
Expected: FAIL (compilation errors, types and methods undefined).

- [ ] **Step 3: Implement Config, Options, APIError, and Client**

Create `config.go`:
```go
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
```

Create `options.go`:
```go
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
```

Create `error.go`:
```go
package wacloudapi

import (
	"fmt"
)

type GraphErrorWrapper struct {
	Error *APIError `json:"error"`
}

type APIError struct {
	HTTPStatusCode int    `json:"-"`
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           int    `json:"code"`
	ErrorSubcode   int    `json:"error_subcode"`
	ErrorUserTitle string `json:"error_user_title,omitempty"`
	ErrorUserMsg   string `json:"error_user_msg,omitempty"`
	FBTraceID      string `json:"fbtrace_id,omitempty"`
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.ErrorUserTitle != "" || e.ErrorUserMsg != "" {
		return fmt.Sprintf("wacloudapi: %s (code: %d, subcode: %d, status: %d) - %s: %s",
			e.Message, e.Code, e.ErrorSubcode, e.HTTPStatusCode, e.ErrorUserTitle, e.ErrorUserMsg)
	}
	return fmt.Sprintf("wacloudapi: %s (code: %d, subcode: %d, status: %d, trace: %s)",
		e.Message, e.Code, e.ErrorSubcode, e.HTTPStatusCode, e.FBTraceID)
}

func (e *APIError) IsRateLimit() bool {
	if e == nil {
		return false
	}
	return e.HTTPStatusCode == 429 || e.Code == 80007 || e.Code == 130429 || e.Code == 613
}

func (e *APIError) IsAuthError() bool {
	if e == nil {
		return false
	}
	return e.HTTPStatusCode == 401 || e.Code == 190 || e.Code == 102
}

func (e *APIError) IsTemplateError() bool {
	if e == nil {
		return false
	}
	return e.Code >= 132000 && e.Code <= 132999
}
```

Create `client.go`:
```go
package wacloudapi

type Client struct {
	config *Config
}

func New(accessToken, phoneNumberID string, opts ...Option) *Client {
	cfg := defaultConfig(accessToken, phoneNumberID)
	for _, opt := range opts {
		opt(cfg)
	}

	c := &Client{
		config: cfg,
	}

	return c
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config.go options.go error.go client.go error_test.go client_test.go
git commit -m "feat: add core client, options, config, and structured APIError"
```

---

### Task 2: Internal HTTP Request Runner with Retries & Error Parsing

**Files:**
- Create: `request.go`
- Test: `request_test.go`

**Interfaces:**
- Consumes: `Client`, `Config`, `APIError` from Task 1
- Produces:
  - `(c *Client) sendJSON(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error`
  - `(c *Client) buildURL(endpoint string) string`

- [ ] **Step 1: Write failing tests for request runner and retry backoff**

Create `request_test.go`:
```go
package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendJSONSuccess(t *testing.T) {
	type dummyRes struct {
		Success bool `json:"success"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(dummyRes{Success: true})
	}))
	defer ts.Close()

	c := New("test-token", "123", WithBaseURL(ts.URL), WithAPIVersion(""))
	var res dummyRes
	err := c.sendJSON(context.Background(), http.MethodPost, "test-endpoint", map[string]string{"foo": "bar"}, &res)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success true")
	}
}

func TestSendJSONGraphAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(GraphErrorWrapper{
			Error: &APIError{
				Message:   "Invalid parameter",
				Type:      "OAuthException",
				Code:      100,
				FBTraceID: "trace123",
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "123", WithBaseURL(ts.URL), WithAPIVersion(""))
	err := c.sendJSON(context.Background(), http.MethodPost, "test-endpoint", nil, nil)
	if err == nil {
		t.Fatalf("expected API error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != 100 || apiErr.HTTPStatusCode != 400 || apiErr.FBTraceID != "trace123" {
		t.Errorf("unexpected APIError fields: %+v", apiErr)
	}
}

func TestSendJSONRetryOnRateLimit(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(GraphErrorWrapper{
				Error: &APIError{Message: "Rate limited", Code: 80007},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	c := New("test-token", "123",
		WithBaseURL(ts.URL),
		WithAPIVersion(""),
		WithRetry(3, 10*time.Millisecond),
	)

	var res map[string]string
	err := c.sendJSON(context.Background(), http.MethodGet, "test-retry", nil, &res)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestSendJSON ./...`
Expected: FAIL (methods `sendJSON` undefined).

- [ ] **Step 3: Implement internal request runner & retry logic**

Create `request.go`:
```go
package wacloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *Client) buildURL(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "/")
	if c.config.APIVersion != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.config.BaseURL, "/"), c.config.APIVersion, endpoint)
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(c.config.BaseURL, "/"), endpoint)
}

func (c *Client) sendJSON(ctx context.Context, method, endpoint string, reqBody interface{}, result interface{}) error {
	var bodyBytes []byte
	var err error
	if reqBody != nil {
		bodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("wacloudapi: failed to marshal request body: %w", err)
		}
	}

	url := c.buildURL(endpoint)

	var resp *http.Response
	maxAttempts := 1 + c.config.MaxRetries

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var reqReader io.Reader
		if len(bodyBytes) > 0 {
			reqReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqReader)
		if err != nil {
			return fmt.Errorf("wacloudapi: failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err = c.config.HTTPClient.Do(req)
		if err != nil {
			// If context is canceled, return immediately
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Transient network error retry
			if attempt < maxAttempts {
				c.backoff(ctx, attempt)
				continue
			}
			return fmt.Errorf("wacloudapi: http request failed: %w", err)
		}

		// Read response body
		respBytes, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("wacloudapi: failed to read response body: %w", readErr)
		}

		// Success check (2xx)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && len(respBytes) > 0 {
				if err := json.Unmarshal(respBytes, result); err != nil {
					return fmt.Errorf("wacloudapi: failed to unmarshal response: %w", err)
				}
			}
			return nil
		}

		// Parse error payload
		apiErr := c.parseAPIError(resp.StatusCode, respBytes)

		// Check if retryable (429 or 5xx)
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < maxAttempts {
			c.backoff(ctx, attempt)
			continue
		}

		return apiErr
	}

	return fmt.Errorf("wacloudapi: request failed after %d attempts", maxAttempts)
}

func (c *Client) parseAPIError(statusCode int, body []byte) *APIError {
	var wrapper GraphErrorWrapper
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Error != nil {
		wrapper.Error.HTTPStatusCode = statusCode
		return wrapper.Error
	}

	return &APIError{
		HTTPStatusCode: statusCode,
		Message:        string(body),
	}
}

func (c *Client) backoff(ctx context.Context, attempt int) {
	wait := c.config.RetryWaitMin * (1 << (attempt - 1))
	if wait > c.config.RetryWaitMax {
		wait = c.config.RetryWaitMax
	}
	select {
	case <-time.After(wait):
	case <-ctx.Done():
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestSendJSON ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add request.go request_test.go
git commit -m "feat: implement internal HTTP request runner, JSON serializer, and exponential backoff retry"
```

---

### Task 3: Messages Models & Messages Service

**Files:**
- Create: `message_models.go`
- Create: `messages.go`
- Modify: `client.go` (attach `client.Messages`)
- Test: `messages_test.go`

**Interfaces:**
- Consumes: `Client`, `sendJSON` from Tasks 1 & 2
- Produces:
  - Models: `SendMessageRequest`, `SendMessageResponse`, `TextMessage`, `MediaMessage`, `TemplateMessage`, `InteractiveMessage`, `Location`, `Contact`, `Reaction`, `MediaSource`, `MediaByID(string)`, `MediaByURL(string)`
  - Message Options: `WithPreviewURL(bool)`, `WithReplyTo(string)`, `WithCaption(string)`, `WithFilename(string)`
  - Methods on `MessagesService`:
    - `Send(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)`
    - `SendText(ctx, to, text, ...)`
    - `SendImage(ctx, to, media, ...)`
    - `SendAudio(ctx, to, media)`
    - `SendVideo(ctx, to, media, ...)`
    - `SendDocument(ctx, to, media, ...)`
    - `SendSticker(ctx, to, media)`
    - `SendLocation(ctx, to, loc, ...)`
    - `SendContacts(ctx, to, contacts, ...)`
    - `SendReaction(ctx, to, messageID, emoji)`
    - `SendTemplate(ctx, to, tpl, ...)`
    - `SendInteractive(ctx, to, interactive, ...)`
    - `MarkAsRead(ctx, messageID)`

- [ ] **Step 1: Write failing tests for MessagesService methods**

Create `messages_test.go`:
```go
package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessagesServiceSendText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/12345/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode req body: %v", err)
		}

		if req.To != "628123456789" || req.Type != "text" || req.Text == nil || req.Text.Body != "Hello WhatsApp" {
			t.Errorf("unexpected request payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status"`
			}{
				{ID: "wamid.test123", MessageStatus: "accepted"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendText(context.Background(), "628123456789", "Hello WhatsApp", WithPreviewURL(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Messages) == 0 || res.Messages[0].ID != "wamid.test123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceSendImageWithMediaID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "image" || req.Image == nil || req.Image.ID != "media-id-99" || req.Image.Caption != "Test caption" {
			t.Errorf("unexpected image payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status"`
			}{
				{ID: "wamid.media123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendImage(context.Background(), "628123456789", MediaByID("media-id-99"), WithCaption("Test caption"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.media123" {
		t.Errorf("unexpected message id: %s", res.Messages[0].ID)
	}
}

func TestMessagesServiceSendReaction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "reaction" || req.Reaction == nil || req.Reaction.MessageID != "wamid.123" || req.Reaction.Emoji != "👍" {
			t.Errorf("unexpected reaction payload: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(SendMessageResponse{
			MessagingProduct: "whatsapp",
			Messages: []struct {
				ID            string `json:"id"`
				MessageStatus string `json:"message_status"`
			}{
				{ID: "wamid.reaction123"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Messages.SendReaction(context.Background(), "628123456789", "wamid.123", "👍")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Messages[0].ID != "wamid.reaction123" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestMessagesServiceMarkAsRead(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["messaging_product"] != "whatsapp" || body["status"] != "read" || body["message_id"] != "wamid.msg123" {
			t.Errorf("unexpected mark as read body: %+v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	err := c.Messages.MarkAsRead(context.Background(), "wamid.msg123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestMessagesService ./...`
Expected: FAIL.

- [ ] **Step 3: Implement message models, options, and MessagesService**

Create `message_models.go`:
```go
package wacloudapi

type MediaSource struct {
	ID   string
	Link string
}

func MediaByID(id string) MediaSource {
	return MediaSource{ID: id}
}

func MediaByURL(url string) MediaSource {
	return MediaSource{Link: url}
}

type MessageOption func(*messageOptions)

type messageOptions struct {
	PreviewURL bool
	ReplyTo    string
	Caption    string
	Filename   string
}

func WithPreviewURL(preview bool) MessageOption {
	return func(o *messageOptions) {
		o.PreviewURL = preview
	}
}

func WithReplyTo(messageID string) MessageOption {
	return func(o *messageOptions) {
		o.ReplyTo = messageID
	}
}

func WithCaption(caption string) MessageOption {
	return func(o *messageOptions) {
		o.Caption = caption
	}
}

func WithFilename(filename string) MessageOption {
	return func(o *messageOptions) {
		o.Filename = filename
	}
}

type MessageContext struct {
	MessageID string `json:"message_id,omitempty"`
}

type TextMessage struct {
	PreviewURL bool   `json:"preview_url,omitempty"`
	Body       string `json:"body"`
}

type MediaMessage struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
}

type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

type ContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type ContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type Contact struct {
	Name      ContactName      `json:"name"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Org       *ContactOrg      `json:"org,omitempty"`
}

type Reaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// Template Models
type TemplateParameter struct {
	Type     string        `json:"type"` // text, currency, date_time, image, document, video
	Text     string        `json:"text,omitempty"`
	Currency *Currency     `json:"currency,omitempty"`
	DateTime *DateTime     `json:"date_time,omitempty"`
	Image    *MediaMessage `json:"image,omitempty"`
	Document *MediaMessage `json:"document,omitempty"`
	Video    *MediaMessage `json:"video,omitempty"`
}

type Currency struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"`
}

type DateTime struct {
	FallbackValue string `json:"fallback_value"`
}

type TemplateComponent struct {
	Type       string              `json:"type"` // header, body, button
	SubType    string              `json:"sub_type,omitempty"` // quick_reply, url
	Index      int                 `json:"index,omitempty"`
	Parameters []TemplateParameter `json:"parameters,omitempty"`
}

type TemplateLanguage struct {
	Code string `json:"code"`
}

type TemplateMessage struct {
	Name       string              `json:"name"`
	Language   TemplateLanguage    `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

// Interactive Models
type InteractiveType string

const (
	InteractiveTypeButton      InteractiveType = "button"
	InteractiveTypeList        InteractiveType = "list"
	InteractiveTypeCTAURL      InteractiveType = "cta_url"
)

type InteractiveHeader struct {
	Type     string        `json:"type"` // text, image, video, document
	Text     string        `json:"text,omitempty"`
	Image    *MediaMessage `json:"image,omitempty"`
	Video    *MediaMessage `json:"video,omitempty"`
	Document *MediaMessage `json:"document,omitempty"`
}

type InteractiveBody struct {
	Text string `json:"text"`
}

type InteractiveFooter struct {
	Text string `json:"text"`
}

type ButtonAction struct {
	Type  string      `json:"type"` // reply
	Reply ButtonReply `json:"reply"`
}

type ButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListSection struct {
	Title string    `json:"title,omitempty"`
	Rows  []ListRow `json:"rows"`
}

type ListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type InteractiveAction struct {
	Button      string         `json:"button,omitempty"`       // Untuk list message (label button utama)
	Buttons     []ButtonAction `json:"buttons,omitempty"`      // Untuk quick reply buttons
	Sections    []ListSection  `json:"sections,omitempty"`     // Untuk list messages
	Name        string         `json:"name,omitempty"`         // Untuk CTA URL ("cta_url")
	Parameters  interface{}    `json:"parameters,omitempty"`   // Parameter CTA URL
}

type InteractiveMessage struct {
	Type   InteractiveType    `json:"type"`
	Header *InteractiveHeader `json:"header,omitempty"`
	Body   InteractiveBody    `json:"body"`
	Footer *InteractiveFooter `json:"footer,omitempty"`
	Action InteractiveAction  `json:"action"`
}

// SendMessageRequest & Response
type SendMessageRequest struct {
	MessagingProduct string              `json:"messaging_product"`
	RecipientType    string              `json:"recipient_type,omitempty"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Context          *MessageContext     `json:"context,omitempty"`
	Text             *TextMessage        `json:"text,omitempty"`
	Image            *MediaMessage       `json:"image,omitempty"`
	Audio            *MediaMessage       `json:"audio,omitempty"`
	Video            *MediaMessage       `json:"video,omitempty"`
	Document         *MediaMessage       `json:"document,omitempty"`
	Sticker          *MediaMessage       `json:"sticker,omitempty"`
	Location         *Location           `json:"location,omitempty"`
	Contacts         []Contact           `json:"contacts,omitempty"`
	Reaction         *Reaction           `json:"reaction,omitempty"`
	Template         *TemplateMessage    `json:"template,omitempty"`
	Interactive      *InteractiveMessage `json:"interactive,omitempty"`
}

type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID            string `json:"id"`
		MessageStatus string `json:"message_status,omitempty"`
	} `json:"messages"`
}
```

Create `messages.go`:
```go
package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
)

type MessagesService struct {
	client *Client
}

func newMessagesService(client *Client) *MessagesService {
	return &MessagesService{client: client}
}

func (s *MessagesService) Send(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: request cannot be nil")
	}
	req.MessagingProduct = "whatsapp"

	endpoint := fmt.Sprintf("%s/messages", s.client.config.PhoneNumberID)
	var resp SendMessageResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *MessagesService) SendText(ctx context.Context, to, text string, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "text",
		Text: &TextMessage{
			Body:       text,
			PreviewURL: opt.PreviewURL,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendImage(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	msg := &MediaMessage{
		ID:      media.ID,
		Link:    media.Link,
		Caption: opt.Caption,
	}
	req := &SendMessageRequest{
		To:    to,
		Type:  "image",
		Image: msg,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendAudio(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "audio",
		Audio: &MediaMessage{
			ID:   media.ID,
			Link: media.Link,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendVideo(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "video",
		Video: &MediaMessage{
			ID:      media.ID,
			Link:    media.Link,
			Caption: opt.Caption,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendDocument(ctx context.Context, to string, media MediaSource, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:   to,
		Type: "document",
		Document: &MediaMessage{
			ID:       media.ID,
			Link:     media.Link,
			Caption:  opt.Caption,
			Filename: opt.Filename,
		},
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendSticker(ctx context.Context, to string, media MediaSource) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "sticker",
		Sticker: &MediaMessage{
			ID:   media.ID,
			Link: media.Link,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendLocation(ctx context.Context, to string, loc Location, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "location",
		Location: &loc,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendContacts(ctx context.Context, to string, contacts []Contact, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "contacts",
		Contacts: contacts,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendReaction(ctx context.Context, to, messageID, emoji string) (*SendMessageResponse, error) {
	req := &SendMessageRequest{
		To:   to,
		Type: "reaction",
		Reaction: &Reaction{
			MessageID: messageID,
			Emoji:     emoji,
		},
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendTemplate(ctx context.Context, to string, tpl *TemplateMessage, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:       to,
		Type:     "template",
		Template: tpl,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) SendInteractive(ctx context.Context, to string, interactive *InteractiveMessage, opts ...MessageOption) (*SendMessageResponse, error) {
	opt := applyMessageOptions(opts...)
	req := &SendMessageRequest{
		To:          to,
		Type:        "interactive",
		Interactive: interactive,
	}
	if opt.ReplyTo != "" {
		req.Context = &MessageContext{MessageID: opt.ReplyTo}
	}
	return s.Send(ctx, req)
}

func (s *MessagesService) MarkAsRead(ctx context.Context, messageID string) error {
	endpoint := fmt.Sprintf("%s/messages", s.client.config.PhoneNumberID)
	body := map[string]string{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}

func applyMessageOptions(opts ...MessageOption) *messageOptions {
	o := &messageOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
```

Update `client.go` to attach `Messages`:
```go
package wacloudapi

type Client struct {
	config   *Config
	Messages *MessagesService
	Media    *MediaService
}

func New(accessToken, phoneNumberID string, opts ...Option) *Client {
	cfg := defaultConfig(accessToken, phoneNumberID)
	for _, opt := range opts {
		opt(cfg)
	}

	c := &Client{
		config: cfg,
	}
	c.Messages = newMessagesService(c)
	c.Media = newMediaService(c)

	return c
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestMessagesService ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add message_models.go messages.go client.go messages_test.go
git commit -m "feat: implement message models and MessagesService with helper methods"
```

---

### Task 4: Media Models, Streaming Upload/Download, and Media Service

**Files:**
- Create: `media_models.go`
- Create: `media.go`
- Test: `media_test.go`

**Interfaces:**
- Consumes: `Client`, `Config`, `sendJSON`, `buildURL` from Tasks 1 & 2
- Produces:
  - Models: `UploadMediaResponse`, `MediaMetadata`
  - Methods on `MediaService`:
    - `Upload(ctx context.Context, filename string, r io.Reader, mimeType string) (*UploadMediaResponse, error)`
    - `UploadFile(ctx context.Context, filePath, mimeType string) (*UploadMediaResponse, error)`
    - `Get(ctx context.Context, mediaID string) (*MediaMetadata, error)`
    - `Download(ctx context.Context, mediaID string) (io.ReadCloser, *MediaMetadata, error)`
    - `DownloadBytes(ctx context.Context, mediaID string) ([]byte, *MediaMetadata, error)`
    - `Delete(ctx context.Context, mediaID string) error`

- [ ] **Step 1: Write failing tests for MediaService (Upload, Get, Download, Delete)**

Create `media_test.go`:
```go
package wacloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMediaServiceUpload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart content type, got %s", r.Header.Get("Content-Type"))
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		if r.FormValue("messaging_product") != "whatsapp" {
			t.Errorf("expected messaging_product whatsapp, got %s", r.FormValue("messaging_product"))
		}
		if r.FormValue("type") != "image/png" {
			t.Errorf("expected type image/png, got %s", r.FormValue("type"))
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get form file: %v", err)
		}
		defer file.Close()

		if handler.Filename != "sample.png" {
			t.Errorf("expected filename sample.png, got %s", handler.Filename)
		}

		_ = json.NewEncoder(w).Encode(UploadMediaResponse{ID: "media-upload-123"})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	dummyData := bytes.NewReader([]byte("fake image data"))
	res, err := c.Media.Upload(context.Background(), "sample.png", dummyData, "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "media-upload-123" {
		t.Errorf("expected id media-upload-123, got %s", res.ID)
	}
}

func TestMediaServiceGetAndDownload(t *testing.T) {
	var downloadServer *httptest.Server
	downloadServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("binary file content"))
	}))
	defer downloadServer.Close()

	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/media-id-777" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(MediaMetadata{
			ID:               "media-id-777",
			URL:              downloadServer.URL,
			MimeType:         "image/jpeg",
			FileSize:         19,
			MessagingProduct: "whatsapp",
		})
	}))
	defer metaServer.Close()

	c := New("test-token", "12345", WithBaseURL(metaServer.URL))
	meta, err := c.Media.Get(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error getting meta: %v", err)
	}
	if meta.FileSize != 19 || meta.MimeType != "image/jpeg" {
		t.Errorf("unexpected meta: %+v", meta)
	}

	data, downloadedMeta, err := c.Media.DownloadBytes(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error downloading bytes: %v", err)
	}
	if string(data) != "binary file content" {
		t.Errorf("unexpected downloaded data: %s", string(data))
	}
	if downloadedMeta.ID != "media-id-777" {
		t.Errorf("unexpected metadata from download: %+v", downloadedMeta)
	}
}

func TestMediaServiceDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	err := c.Media.Delete(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error deleting media: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestMediaService ./...`
Expected: FAIL.

- [ ] **Step 3: Implement Media models and MediaService**

Create `media_models.go`:
```go
package wacloudapi

type UploadMediaResponse struct {
	ID string `json:"id"`
}

type MediaMetadata struct {
	ID               string `json:"id"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	MessagingProduct string `json:"messaging_product"`
}
```

Create `media.go`:
```go
package wacloudapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type MediaService struct {
	client *Client
}

func newMediaService(client *Client) *MediaService {
	return &MediaService{client: client}
}

func (s *MediaService) Upload(ctx context.Context, filename string, r io.Reader, mimeType string) (*UploadMediaResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("wacloudapi: media reader cannot be nil")
	}

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	if err := writer.WriteField("messaging_product", "whatsapp"); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to write multipart field: %w", err)
	}
	if err := writer.WriteField("type", mimeType); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to write type field: %w", err)
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to close multipart writer: %w", err)
	}

	endpoint := fmt.Sprintf("%s/media", s.client.config.PhoneNumberID)
	url := s.client.buildURL(endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyBuf)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to create upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.client.config.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: media upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to read upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, s.client.parseAPIError(resp.StatusCode, respBytes)
	}

	var uploadResp UploadMediaResponse
	if err := jsonUnmarshal(respBytes, &uploadResp); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to unmarshal upload response: %w", err)
	}

	return &uploadResp, nil
}

func (s *MediaService) UploadFile(ctx context.Context, filePath, mimeType string) (*UploadMediaResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	filename := filepath.Base(filePath)
	return s.Upload(ctx, filename, file, mimeType)
}

func (s *MediaService) Get(ctx context.Context, mediaID string) (*MediaMetadata, error) {
	if mediaID == "" {
		return nil, fmt.Errorf("wacloudapi: mediaID cannot be empty")
	}

	endpoint := mediaID
	var meta MediaMetadata
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *MediaService) Download(ctx context.Context, mediaID string) (io.ReadCloser, *MediaMetadata, error) {
	meta, err := s.Get(ctx, mediaID)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meta.URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: failed to create download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.client.config.AccessToken)

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: media download failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, nil, s.client.parseAPIError(resp.StatusCode, respBytes)
	}

	return resp.Body, meta, nil
}

func (s *MediaService) DownloadBytes(ctx context.Context, mediaID string) ([]byte, *MediaMetadata, error) {
	stream, meta, err := s.Download(ctx, mediaID)
	if err != nil {
		return nil, nil, err
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: failed to read downloaded media bytes: %w", err)
	}
	return data, meta, nil
}

func (s *MediaService) Delete(ctx context.Context, mediaID string) error {
	if mediaID == "" {
		return fmt.Errorf("wacloudapi: mediaID cannot be empty")
	}
	endpoint := mediaID
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
```
*(Catatan: pastikan `import "encoding/json"` ditambahkan di `media.go`)*

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestMediaService ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add media_models.go media.go media_test.go
git commit -m "feat: implement MediaService (Upload multipart, Get metadata, Download streaming, and Delete)"
```

---

### Task 5: Webhook Models, Signature Verification & Standalone Parser

**Files:**
- Create: `webhook/models.go`
- Create: `webhook/webhook.go`
- Test: `webhook/webhook_test.go`

**Interfaces:**
- Produces:
  - Models: `Payload`, `Entry`, `Change`, `Value`, `Metadata`, `Contact`, `Message`, `Status`, `Conversation`, `Pricing`, `Origin`, `WebhookError`, `Text`, `Media`, `Location`, `Interactive`, `Reaction`
  - Functions:
    - `VerifyChallenge(w http.ResponseWriter, r *http.Request, verifyToken string) bool`
    - `VerifySignature(body []byte, signatureHeader string, appSecret string) error`
    - `ParsePayload(body []byte) (*Payload, error)`

- [ ] **Step 1: Write failing tests for Challenge & HMAC Signature Verification, and JSON payload parsing**

Create `webhook/webhook_test.go`:
```go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyChallenge(t *testing.T) {
	verifyToken := "my-secret-token"

	// Valid challenge
	req := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=my-secret-token&hub.challenge=challenge123", nil)
	w := httptest.NewRecorder()
	if !VerifyChallenge(w, req, verifyToken) {
		t.Errorf("expected VerifyChallenge to succeed")
	}
	if w.Code != http.StatusOK || w.Body.String() != "challenge123" {
		t.Errorf("unexpected response: code %d, body: %s", w.Code, w.Body.String())
	}

	// Invalid token
	reqInvalid := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=challenge123", nil)
	wInvalid := httptest.NewRecorder()
	if VerifyChallenge(wInvalid, reqInvalid, verifyToken) {
		t.Errorf("expected VerifyChallenge to fail on invalid token")
	}
	if wInvalid.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", wInvalid.Code)
	}
}

func TestVerifySignature(t *testing.T) {
	appSecret := "meta-app-secret-xyz"
	body := []byte(`{"object":"whatsapp_business_account"}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := VerifySignature(body, validSig, appSecret); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}

	if err := VerifySignature(body, "sha256=invalidhex", appSecret); err == nil {
		t.Errorf("expected error on invalid signature")
	}

	if err := VerifySignature(body, "", appSecret); err == nil {
		t.Errorf("expected error on empty signature header")
	}
}

func TestParsePayloadTextMessage(t *testing.T) {
	rawJSON := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "WABA_ID_123",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "15550234567",
						"phone_number_id": "1234567890"
					},
					"contacts": [{
						"profile": { "name": "Alice" },
						"wa_id": "62811111111"
					}],
					"messages": [{
						"from": "62811111111",
						"id": "wamid.HBgM...",
						"timestamp": "1700000000",
						"type": "text",
						"text": { "body": "Halo, saya butuh bantuan" }
					}]
				}
			}]
		}]
	}`)

	payload, err := ParsePayload(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing payload: %v", err)
	}

	if payload.Object != "whatsapp_business_account" || len(payload.Entry) == 0 {
		t.Fatalf("unexpected payload structure: %+v", payload)
	}

	msg := payload.Entry[0].Changes[0].Value.Messages[0]
	if msg.From != "62811111111" || msg.Type != "text" || msg.Text == nil || msg.Text.Body != "Halo, saya butuh bantuan" {
		t.Errorf("unexpected message content: %+v", msg)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v ./webhook/...`
Expected: FAIL.

- [ ] **Step 3: Implement Webhook Models, Challenge Verification, Signature Verifier, and JSON Parser**

Create `webhook/models.go`:
```go
package webhook

type Payload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string `json:"field"`
	Value Value  `json:"value"`
}

type Value struct {
	MessagingProduct string     `json:"messaging_product"`
	Metadata         Metadata   `json:"metadata"`
	Contacts         []Contact  `json:"contacts,omitempty"`
	Messages         []Message  `json:"messages,omitempty"`
	Statuses         []Status   `json:"statuses,omitempty"`
	Errors           []APIError `json:"errors,omitempty"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	Profile ContactProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

type ContactProfile struct {
	Name string `json:"name"`
}

type Message struct {
	From        string       `json:"from"`
	ID          string       `json:"id"`
	Timestamp   string       `json:"timestamp"`
	Type        string       `json:"type"` // text, image, audio, video, document, sticker, location, contacts, interactive, reaction, system, unsupported
	Context     *Context     `json:"context,omitempty"`
	Text        *Text        `json:"text,omitempty"`
	Image       *Media       `json:"image,omitempty"`
	Audio       *Media       `json:"audio,omitempty"`
	Video       *Media       `json:"video,omitempty"`
	Document    *Media       `json:"document,omitempty"`
	Sticker     *Media       `json:"sticker,omitempty"`
	Location    *Location    `json:"location,omitempty"`
	Contacts    []Contact    `json:"contacts,omitempty"`
	Interactive *Interactive `json:"interactive,omitempty"`
	Reaction    *Reaction    `json:"reaction,omitempty"`
	System      *System      `json:"system,omitempty"`
}

func (m *Message) MediaID() string {
	switch m.Type {
	case "image":
		if m.Image != nil {
			return m.Image.ID
		}
	case "audio":
		if m.Audio != nil {
			return m.Audio.ID
		}
	case "video":
		if m.Video != nil {
			return m.Video.ID
		}
	case "document":
		if m.Document != nil {
			return m.Document.ID
		}
	case "sticker":
		if m.Sticker != nil {
			return m.Sticker.ID
		}
	}
	return ""
}

type Context struct {
	From        string `json:"from,omitempty"`
	ID          string `json:"id,omitempty"`
	Forwarded   bool   `json:"forwarded,omitempty"`
	FrequentlyForwarded bool `json:"frequently_forwarded,omitempty"`
}

type Text struct {
	Body string `json:"body"`
}

type Media struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type Reaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type Interactive struct {
	Type        string            `json:"type"` // button_reply, list_reply
	ButtonReply *ButtonReplyValue `json:"button_reply,omitempty"`
	ListReply   *ListReplyValue   `json:"list_reply,omitempty"`
}

const (
	InteractiveTypeButtonReply = "button_reply"
	InteractiveTypeListReply   = "list_reply"
)

type ButtonReplyValue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListReplyValue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type System struct {
	Body     string `json:"body"`
	Identity string `json:"identity,omitempty"`
	WaID     string `json:"wa_id,omitempty"`
	Type     string `json:"type,omitempty"`
	Customer string `json:"customer,omitempty"`
}

type Status struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"` // sent, delivered, read, failed
	Timestamp    string        `json:"timestamp"`
	RecipientID  string        `json:"recipient_id"`
	Conversation *Conversation `json:"conversation,omitempty"`
	Pricing      *Pricing      `json:"pricing,omitempty"`
	Errors       []APIError    `json:"errors,omitempty"`
}

type Conversation struct {
	ID                  string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Origin              Origin `json:"origin"`
}

type Origin struct {
	Type string `json:"type"` // user_initiated, business_initiated, referral_conversion
}

type Pricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

type APIError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message,omitempty"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data,omitempty"`
}
```

Create `webhook/webhook.go`:
```go
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrMissingSignature = errors.New("webhook: missing X-Hub-Signature-256 header")
	ErrInvalidSignature = errors.New("webhook: signature mismatch (invalid X-Hub-Signature-256)")
)

func VerifyChallenge(w http.ResponseWriter, r *http.Request, verifyToken string) bool {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == verifyToken {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return true
	}

	w.WriteHeader(http.StatusForbidden)
	return false
}

func VerifySignature(body []byte, signatureHeader string, appSecret string) error {
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return ErrInvalidSignature
	}
	expectedSigHex := parts[1]

	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return fmt.Errorf("webhook: failed to decode signature hex: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	actualSig := mac.Sum(nil)

	if !hmac.Equal(expectedSig, actualSig) {
		return ErrInvalidSignature
	}

	return nil
}

func ParsePayload(body []byte) (*Payload, error) {
	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("webhook: failed to parse payload JSON: %w", err)
	}
	return &payload, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./webhook/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webhook/models.go webhook/webhook.go webhook/webhook_test.go
git commit -m "feat: implement Webhook models, challenge verification, and HMAC-SHA256 signature verifier"
```

---

### Task 6: Webhook HTTP Event Handler & Dispatcher

**Files:**
- Create: `webhook/handler.go`
- Test: `webhook/handler_test.go`

**Interfaces:**
- Consumes: `Payload`, `Message`, `Status`, `Metadata`, `VerifyChallenge`, `VerifySignature`, `ParsePayload` from Task 5
- Produces:
  - `Handler`, `NewHandler(verifyToken, appSecret string) *Handler`
  - `(h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)`
  - Callback Registrations:
    - `OnMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
    - `OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
    - `OnMediaMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
    - `OnInteractiveMessage(func(ctx context.Context, msg Message, meta Metadata) error)`
    - `OnStatus(func(ctx context.Context, status Status, meta Metadata) error)`
    - `OnError(func(ctx context.Context, err error))`

- [ ] **Step 1: Write failing tests for Webhook Handler Event Dispatching**

Create `webhook/handler_test.go`:
```go
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestHandlerServeHTTP(t *testing.T) {
	verifyToken := "test-verify-token"
	appSecret := "test-secret"
	h := NewHandler(verifyToken, appSecret)

	var textReceived int32
	var statusReceived int32

	h.OnTextMessage(func(ctx context.Context, msg Message, meta Metadata) error {
		if msg.Text.Body == "Hello Bot" && meta.PhoneNumberID == "123456" {
			atomic.AddInt32(&textReceived, 1)
		}
		return nil
	})

	h.OnStatus(func(ctx context.Context, status Status, meta Metadata) error {
		if status.Status == "delivered" {
			atomic.AddInt32(&statusReceived, 1)
		}
		return nil
	})

	// 1. Test GET Challenge Verification
	reqGet := httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token="+verifyToken+"&hub.challenge=test_chall", nil)
	wGet := httptest.NewRecorder()
	h.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK || wGet.Body.String() != "test_chall" {
		t.Errorf("unexpected GET challenge response: code %d, body %s", wGet.Code, wGet.Body.String())
	}

	// 2. Test POST Event Dispatching
	rawPayload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": { "phone_number_id": "123456", "display_phone_number": "123" },
					"messages": [{ "from": "6281", "id": "wamid.1", "type": "text", "text": { "body": "Hello Bot" } }],
					"statuses": [{ "id": "wamid.0", "status": "delivered", "recipient_id": "6281" }]
				}
			}]
		}]
	}`)

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(rawPayload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	reqPost := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(rawPayload))
	reqPost.Header.Set("X-Hub-Signature-256", sig)
	wPost := httptest.NewRecorder()

	h.ServeHTTP(wPost, reqPost)
	if wPost.Code != http.StatusOK {
		t.Errorf("expected POST status 200, got %d", wPost.Code)
	}

	if atomic.LoadInt32(&textReceived) != 1 {
		t.Errorf("expected 1 text message received, got %d", textReceived)
	}
	if atomic.LoadInt32(&statusReceived) != 1 {
		t.Errorf("expected 1 status received, got %d", statusReceived)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestHandlerServeHTTP ./webhook/...`
Expected: FAIL.

- [ ] **Step 3: Implement Webhook Handler & Dispatcher**

Create `webhook/handler.go`:
```go
package webhook

import (
	"context"
	"io"
	"net/http"
	"sync"
)

type MessageHandler func(ctx context.Context, msg Message, meta Metadata) error
type StatusHandler func(ctx context.Context, status Status, meta Metadata) error
type ErrorHandler func(ctx context.Context, err error)

type Handler struct {
	verifyToken string
	appSecret   string

	mu                 sync.RWMutex
	onMessage          []MessageHandler
	onTextMessage      []MessageHandler
	onMediaMessage     []MessageHandler
	onInteractiveMsg   []MessageHandler
	onStatus           []StatusHandler
	onError            []ErrorHandler
}

func NewHandler(verifyToken, appSecret string) *Handler {
	return &Handler{
		verifyToken: verifyToken,
		appSecret:   appSecret,
	}
}

func (h *Handler) OnMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage = append(h.onMessage, fn)
}

func (h *Handler) OnTextMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onTextMessage = append(h.onTextMessage, fn)
}

func (h *Handler) OnMediaMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMediaMessage = append(h.onMediaMessage, fn)
}

func (h *Handler) OnInteractiveMessage(fn MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onInteractiveMsg = append(h.onInteractiveMsg, fn)
}

func (h *Handler) OnStatus(fn StatusHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onStatus = append(h.onStatus, fn)
}

func (h *Handler) OnError(fn ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onError = append(h.onError, fn)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		VerifyChallenge(w, r, h.verifyToken)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.dispatchError(r.Context(), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.appSecret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if err := VerifySignature(bodyBytes, sig, h.appSecret); err != nil {
			h.dispatchError(r.Context(), err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	payload, err := ParsePayload(bodyBytes)
	if err != nil {
		h.dispatchError(r.Context(), err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Dispatch events
	go h.dispatchEvents(context.Background(), payload)
}

func (h *Handler) dispatchEvents(ctx context.Context, payload *Payload) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			val := change.Value
			meta := val.Metadata

			// Process Messages
			for _, msg := range val.Messages {
				for _, fn := range h.onMessage {
					_ = fn(ctx, msg, meta)
				}

				if msg.Type == "text" {
					for _, fn := range h.onTextMessage {
						_ = fn(ctx, msg, meta)
					}
				}

				if isMediaType(msg.Type) {
					for _, fn := range h.onMediaMessage {
						_ = fn(ctx, msg, meta)
					}
				}

				if msg.Type == "interactive" {
					for _, fn := range h.onInteractiveMsg {
						_ = fn(ctx, msg, meta)
					}
				}
			}

			// Process Statuses
			for _, status := range val.Statuses {
				for _, fn := range h.onStatus {
					_ = fn(ctx, status, meta)
				}
			}
		}
	}
}

func (h *Handler) dispatchError(ctx context.Context, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, fn := range h.onError {
		fn(ctx, err)
	}
}

func isMediaType(t string) bool {
	switch t {
	case "image", "audio", "video", "document", "sticker":
		return true
	default:
		return false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./webhook/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webhook/handler.go webhook/handler_test.go
git commit -m "feat: implement Webhook HTTP handler and event dispatcher"
```

---

### Task 7: Examples, Documentation & Full Verification

**Files:**
- Create: `examples/send_messages/main.go`
- Create: `examples/media_upload/main.go`
- Create: `examples/webhook_server/main.go`
- Create: `README.md`

**Interfaces:**
- Consumes: All packages and features implemented in Tasks 1-6

- [ ] **Step 1: Create runnable examples**

Create `examples/send_messages/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/itsmeabde/wacloudapi"
)

func main() {
	token := os.Getenv("WA_ACCESS_TOKEN")
	phoneID := os.Getenv("WA_PHONE_NUMBER_ID")
	recipient := os.Getenv("WA_RECIPIENT_PHONE")

	if token == "" || phoneID == "" || recipient == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_RECIPIENT_PHONE must be set")
	}

	client := wacloudapi.New(token, phoneID, wacloudapi.WithRetry(3))

	// Send Text Message
	res, err := client.Messages.SendText(context.Background(), recipient, "Halo dari wacloudapi Go SDK!")
	if err != nil {
		log.Fatalf("Error sending text: %v", err)
	}
	fmt.Printf("Message sent! ID: %s\n", res.Messages[0].ID)

	// Send Interactive Button Message
	interactive := &wacloudapi.InteractiveMessage{
		Type: wacloudapi.InteractiveTypeButton,
		Body: wacloudapi.InteractiveBody{Text: "Apakah Anda ingin melanjutkan konfirmasi?"},
		Action: wacloudapi.InteractiveAction{
			Buttons: []wacloudapi.ButtonAction{
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_yes", Title: "Ya, Setuju"}},
				{Type: "reply", Reply: wacloudapi.ButtonReply{ID: "btn_no", Title: "Batalkan"}},
			},
		},
	}
	resBtn, err := client.Messages.SendInteractive(context.Background(), recipient, interactive)
	if err != nil {
		log.Fatalf("Error sending interactive: %v", err)
	}
	fmt.Printf("Interactive message sent! ID: %s\n", resBtn.Messages[0].ID)
}
```

Create `examples/media_upload/main.go`:
```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/itsmeabde/wacloudapi"
)

func main() {
	token := os.Getenv("WA_ACCESS_TOKEN")
	phoneID := os.Getenv("WA_PHONE_NUMBER_ID")
	recipient := os.Getenv("WA_RECIPIENT_PHONE")

	if token == "" || phoneID == "" || recipient == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_RECIPIENT_PHONE must be set")
	}

	client := wacloudapi.New(token, phoneID)

	// Upload in-memory buffer
	buf := bytes.NewReader([]byte("dummy content"))
	uploaded, err := client.Media.Upload(context.Background(), "notes.txt", buf, "text/plain")
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}
	fmt.Printf("Media uploaded! ID: %s\n", uploaded.ID)

	// Send Document using Media ID
	res, err := client.Messages.SendDocument(context.Background(), recipient,
		wacloudapi.MediaByID(uploaded.ID),
		wacloudapi.WithCaption("Berikut catatan terlampir"),
		wacloudapi.WithFilename("notes.txt"),
	)
	if err != nil {
		log.Fatalf("Send document failed: %v", err)
	}
	fmt.Printf("Document message sent! ID: %s\n", res.Messages[0].ID)
}
```

Create `examples/webhook_server/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/itsmeabde/wacloudapi/webhook"
)

func main() {
	verifyToken := os.Getenv("WA_VERIFY_TOKEN")
	appSecret := os.Getenv("WA_APP_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	h := webhook.NewHandler(verifyToken, appSecret)

	h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("[Text Message] From: %s, Body: %s\n", msg.From, msg.Text.Body)
		return nil
	})

	h.OnMediaMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		fmt.Printf("[Media Message] From: %s, Type: %s, MediaID: %s\n", msg.From, msg.Type, msg.MediaID())
		return nil
	})

	h.OnInteractiveMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
		if msg.Interactive.ButtonReply != nil {
			fmt.Printf("[Button Reply] From: %s, Title: %s, ID: %s\n", msg.From, msg.Interactive.ButtonReply.Title, msg.Interactive.ButtonReply.ID)
		}
		return nil
	})

	h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
		fmt.Printf("[Status Update] ID: %s, Status: %s, Recipient: %s\n", status.ID, status.Status, status.RecipientID)
		return nil
	})

	h.OnError(func(ctx context.Context, err error) {
		log.Printf("[Webhook Error]: %v\n", err)
	})

	http.Handle("/webhook", h)
	log.Printf("Webhook server running on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
```

Create `README.md`:
```markdown
# Go WhatsApp Cloud API SDK (`wacloudapi`)

[![Go Reference](https://pkg.go.dev/badge/github.com/itsmeabde/wacloudapi.svg)](https://pkg.go.dev/github.com/itsmeabde/wacloudapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsmeabde/wacloudapi)](https://goreportcard.com/report/github.com/itsmeabde/wacloudapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Package Go idiomatik, modular, aman secara konkuren (*thread-safe*), dan **zero external dependencies** untuk berinteraksi dengan **Meta WhatsApp Cloud API (Graph API v21.0+)**.

## ✨ Fitur Utama
- **Zero External Dependencies**: Dibangun 100% menggunakan Go Standard Library (`net/http`, `crypto/hmac`, dll.).
- **Lengkap & Type-Safe**: Mendukung semua jenis pesan WhatsApp (Teks, Media, Template, Interaktif/Tombol/List, Lokasi, Kontak, Reaksi, Quoted Reply).
- **Media Streaming**: Upload multipart dan download streaming hemat memori (`io.Reader` / `io.ReadCloser`).
- **Dual Webhook Processing**: Event dispatcher siap pakai (`http.Handler`) dengan callback bertipe serta standalone validator signature HMAC-SHA256.
- **Resilient & Structured Errors**: Penanganan error terstruktur (`*APIError`), klasifikasi error (`IsRateLimit()`, `IsAuthError()`), serta retry exponential backoff otomatis.

## 📦 Instalasi

```bash
go get github.com/itsmeabde/wacloudapi
```

## 🚀 Quick Start

### Inisialisasi Client
```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/itsmeabde/wacloudapi"
)

func main() {
    client := wacloudapi.New("ACCESS_TOKEN", "PHONE_NUMBER_ID",
        wacloudapi.WithRetry(3),
        wacloudapi.WithTimeout(15 * time.Second),
    )

    // Kirim pesan teks sederhana
    res, err := client.Messages.SendText(context.Background(), "628123456789", "Halo dari wacloudapi!")
    if err != nil {
        panic(err)
    }
    fmt.Println("Pesan terkirim dengan ID:", res.Messages[0].ID)
}
```

### Menangani Webhook
```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "github.com/itsmeabde/wacloudapi/webhook"
)

func main() {
    h := webhook.NewHandler("VERIFY_TOKEN", "APP_SECRET")

    h.OnTextMessage(func(ctx context.Context, msg webhook.Message, meta webhook.Metadata) error {
        fmt.Printf("Pesan dari %s: %s\n", msg.From, msg.Text.Body)
        return nil
    })

    h.OnStatus(func(ctx context.Context, status webhook.Status, meta webhook.Metadata) error {
        fmt.Printf("Status pesan %s: %s\n", status.ID, status.Status)
        return nil
    })

    http.Handle("/webhook", h)
    http.ListenAndServe(":8080", nil)
}
```

## 📚 Lisensi
MIT License.
```

- [ ] **Step 2: Run all tests across the repository**

Run: `go test -v -race -cover ./...`
Expected: ALL PASS with high coverage.

- [ ] **Step 3: Commit**

```bash
git add examples/ README.md
git commit -m "docs: add runnable examples, README documentation, and verify full test suite"
```
