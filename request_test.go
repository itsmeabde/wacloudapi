package wacloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	c := New("test-token", "123", WithBaseURL(ts.URL), WithAPIVersion("v21.0"))
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

	c := New("test-token", "123", WithBaseURL(ts.URL))
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

func TestSendJSONRetryOnMetaRateLimitErrorCode(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusBadRequest) // Status 400 but Meta rate limit error code 130429
			_ = json.NewEncoder(w).Encode(GraphErrorWrapper{
				Error: &APIError{Message: "Rate limit hit", Code: 130429},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	c := New("test-token", "123",
		WithBaseURL(ts.URL),
		WithRetry(2, 5*time.Millisecond),
	)

	var res map[string]string
	err := c.sendJSON(context.Background(), http.MethodGet, "test-retry-meta-code", nil, &res)
	if err != nil {
		t.Fatalf("expected success after retry on Meta rate limit code, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestSendJSONRetryOn5xx(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`Internal Server Error`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"recovered"}`))
	}))
	defer ts.Close()

	c := New("test-token", "123",
		WithBaseURL(ts.URL),
		WithRetry(2, 5*time.Millisecond),
	)

	var res map[string]string
	err := c.sendJSON(context.Background(), http.MethodGet, "test-5xx", nil, &res)
	if err != nil {
		t.Fatalf("expected success after retry on 500, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestSendJSONRetryExhausted(t *testing.T) {
	var attempts int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`Bad Gateway`))
	}))
	defer ts.Close()

	c := New("test-token", "123",
		WithBaseURL(ts.URL),
		WithRetry(2, 5*time.Millisecond),
	)

	err := c.sendJSON(context.Background(), http.MethodGet, "test-exhausted", nil, nil)
	if err == nil {
		t.Fatalf("expected error after retries exhausted, got nil")
	}
	if atomic.LoadInt32(&attempts) != 3 { // 1 initial + 2 retries = 3
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.HTTPStatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", apiErr.HTTPStatusCode)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendJSONNetworkErrorRetry(t *testing.T) {
	var attempts int32
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			count := atomic.AddInt32(&attempts, 1)
			if count == 1 {
				return nil, errors.New("network failure")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioNopCloserString(`{"ok":true}`),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := New("test-token", "123",
		WithHTTPClient(client),
		WithRetry(2, 5*time.Millisecond),
	)

	var res map[string]bool
	err := c.sendJSON(context.Background(), http.MethodGet, "test-net-retry", nil, &res)
	if err != nil {
		t.Fatalf("expected recovery after network retry, got: %v", err)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
	if !res["ok"] {
		t.Errorf("expected ok=true")
	}
}

func TestSendJSONNetworkErrorExhausted(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("persistent network down")
		}),
	}

	c := New("test-token", "123",
		WithHTTPClient(client),
		WithRetry(1, 5*time.Millisecond),
	)

	err := c.sendJSON(context.Background(), http.MethodGet, "test-net-fail", nil, nil)
	if err == nil {
		t.Fatalf("expected network failure error, got nil")
	}
}

func ioNopCloserString(s string) io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString(s))
}

func TestSendJSONInvalidNewRequest(t *testing.T) {
	c := New("test-token", "123")
	// Invalid method with spaces / control chars causes http.NewRequestWithContext to fail
	err := c.sendJSON(context.Background(), "INVALID METHOD\n", "test", nil, nil)
	if err == nil {
		t.Fatalf("expected error from invalid HTTP method, got nil")
	}
}

func TestSendJSONUnmarshalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-a-valid-json"))
	}))
	defer ts.Close()

	c := New("test-token", "123", WithBaseURL(ts.URL))
	var res map[string]string
	err := c.sendJSON(context.Background(), http.MethodGet, "test-unmarshal", nil, &res)
	if err == nil {
		t.Fatalf("expected unmarshal error on invalid JSON response, got nil")
	}
}

func TestSendJSONContextCanceled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	c := New("test-token", "123",
		WithBaseURL(ts.URL),
		WithRetry(5, 500*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := c.sendJSON(ctx, http.MethodGet, "test-cancel", nil, nil)
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestSendJSONMarshalError(t *testing.T) {
	c := New("test-token", "123")
	// Channels cannot be marshaled to JSON
	invalidBody := map[string]interface{}{"ch": make(chan int)}
	err := c.sendJSON(context.Background(), http.MethodPost, "test", invalidBody, nil)
	if err == nil {
		t.Fatalf("expected marshal error, got nil")
	}
}

func TestSendJSONNonJSONResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden Access"))
	}))
	defer ts.Close()

	c := New("test-token", "123", WithBaseURL(ts.URL))
	err := c.sendJSON(context.Background(), http.MethodGet, "test-forbidden", nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.HTTPStatusCode != http.StatusForbidden || apiErr.Message != "Forbidden Access" {
		t.Errorf("unexpected APIError: %+v", apiErr)
	}
}

func TestBackoffCappedAtWaitMax(t *testing.T) {
	c := New("test-token", "123",
		WithRetry(10, 100*time.Millisecond),
	)
	c.config.RetryWaitMax = 50 * time.Millisecond

	start := time.Now()
	// attempt 5 with 100ms min would be 1600ms without cap, but should cap at 50ms
	c.backoff(context.Background(), 5)
	elapsed := time.Since(start)

	if elapsed > 150*time.Millisecond {
		t.Errorf("expected backoff to be capped around 50ms, elapsed: %v", elapsed)
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		apiVersion string
		endpoint   string
		expected   string
	}{
		{
			name:       "standard with version",
			baseURL:    "https://graph.facebook.com",
			apiVersion: "v21.0",
			endpoint:   "123456/messages",
			expected:   "https://graph.facebook.com/v21.0/123456/messages",
		},
		{
			name:       "trailing slash on base and leading slash on endpoint",
			baseURL:    "https://graph.facebook.com/",
			apiVersion: "v21.0",
			endpoint:   "/123456/messages",
			expected:   "https://graph.facebook.com/v21.0/123456/messages",
		},
		{
			name:       "empty api version",
			baseURL:    "https://example.com/api",
			apiVersion: "",
			endpoint:   "endpoint",
			expected:   "https://example.com/api/endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{
				config: &Config{
					BaseURL:    tc.baseURL,
					APIVersion: tc.apiVersion,
				},
			}
			url := c.buildURL(tc.endpoint)
			if url != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, url)
			}
		})
	}
}
