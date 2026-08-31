package wacloudapitest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/itsmeabde/wacloudapi"
	"github.com/itsmeabde/wacloudapi/wacloudapitest"
)

func TestServer_InjectRateLimit_RetryRecovery(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject 1 rate limit error
	srv.InjectRateLimit(1)

	// Create client with 2 retries and short backoff for fast testing
	client := wacloudapi.New(
		srv.AccessToken(),
		srv.PhoneNumberID(),
		wacloudapi.WithBaseURL(srv.URL()),
		wacloudapi.WithAPIVersion(srv.APIVersion()),
		wacloudapi.WithHTTPClient(srv.HTTPClient()),
		wacloudapi.WithRetry(2, 5*time.Millisecond),
	)

	ctx := context.Background()
	resp, err := client.Messages.SendText(ctx, "1234567890", "Hello with retry")
	if err != nil {
		t.Fatalf("expected message send to succeed after retry, got error: %v", err)
	}

	if resp == nil || len(resp.Messages) == 0 {
		t.Fatalf("expected non-empty message response, got %+v", resp)
	}

	if len(srv.SentMessages()) != 1 {
		t.Fatalf("expected 1 sent message recorded, got %d", len(srv.SentMessages()))
	}
}

func TestServer_InjectRateLimit_ExceedingRetries(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject 3 rate limit errors, but client only retries 1 time (2 attempts total)
	srv.InjectRateLimit(3)

	client := wacloudapi.New(
		srv.AccessToken(),
		srv.PhoneNumberID(),
		wacloudapi.WithBaseURL(srv.URL()),
		wacloudapi.WithAPIVersion(srv.APIVersion()),
		wacloudapi.WithHTTPClient(srv.HTTPClient()),
		wacloudapi.WithRetry(1, 5*time.Millisecond),
	)

	ctx := context.Background()
	_, err := client.Messages.SendText(ctx, "1234567890", "Should fail due to rate limit")
	if err == nil {
		t.Fatal("expected error due to exhausted retries, got nil")
	}

	var apiErr *wacloudapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *wacloudapi.APIError, got %T: %v", err, err)
	}

	if !apiErr.IsRateLimit() {
		t.Errorf("expected IsRateLimit() to be true, got code %d, status %d", apiErr.Code, apiErr.HTTPStatusCode)
	}

	if len(srv.SentMessages()) != 0 {
		t.Errorf("expected 0 sent messages recorded, got %d", len(srv.SentMessages()))
	}
}

func TestServer_InjectServerError_RetryRecovery(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject 1 HTTP 500 error
	srv.InjectServerError(1)

	client := wacloudapi.New(
		srv.AccessToken(),
		srv.PhoneNumberID(),
		wacloudapi.WithBaseURL(srv.URL()),
		wacloudapi.WithAPIVersion(srv.APIVersion()),
		wacloudapi.WithHTTPClient(srv.HTTPClient()),
		wacloudapi.WithRetry(2, 5*time.Millisecond),
	)

	ctx := context.Background()
	resp, err := client.Messages.SendText(ctx, "1234567890", "Hello 500 retry")
	if err != nil {
		t.Fatalf("expected send to succeed on retry after 500, got: %v", err)
	}

	if resp == nil || len(resp.Messages) == 0 {
		t.Fatalf("expected non-empty response, got %+v", resp)
	}
}

func TestServer_InjectMetaError_NonRetryable(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject OAuth authentication error (code 190)
	srv.InjectMetaError(190, "Invalid OAuth access token", 1)

	client := srv.Client()
	ctx := context.Background()

	_, err := client.Messages.SendText(ctx, "1234567890", "Will fail with auth error")
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}

	var apiErr *wacloudapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *wacloudapi.APIError, got %T: %v", err, err)
	}

	if !apiErr.IsAuthError() {
		t.Errorf("expected IsAuthError() to be true, got code %d, status %d", apiErr.Code, apiErr.HTTPStatusCode)
	}

	if apiErr.Code != 190 {
		t.Errorf("expected error code 190, got %d", apiErr.Code)
	}

	if apiErr.Message != "Invalid OAuth access token" {
		t.Errorf("expected message %q, got %q", "Invalid OAuth access token", apiErr.Message)
	}
}

func TestServer_InjectMetaError_Variants(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	testCases := []struct {
		code       int
		message    string
		expectedST int
	}{
		{code: 80007, message: "Rate limit", expectedST: http.StatusTooManyRequests},
		{code: 102, message: "Session expired", expectedST: http.StatusUnauthorized},
		{code: 1, message: "Internal unknown", expectedST: http.StatusInternalServerError},
		{code: 100, message: "Invalid parameter", expectedST: http.StatusBadRequest},
		{code: 999, message: "Generic custom error", expectedST: http.StatusBadRequest},
	}

	for _, tc := range testCases {
		srv.InjectMetaError(tc.code, tc.message, 1)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/v21.0/health", nil)
		if err != nil {
			t.Fatalf("failed to create req: %v", err)
		}
		resp, err := srv.HTTPClient().Do(req)
		if err != nil {
			t.Fatalf("failed to exec req: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != tc.expectedST {
			t.Errorf("code %d: expected HTTP status %d, got %d", tc.code, tc.expectedST, resp.StatusCode)
		}
	}
}

func TestServer_Inject_ZeroOrNegativeCount(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Calling with 0 or negative should be no-op
	srv.InjectRateLimit(0)
	srv.InjectRateLimit(-1)
	srv.InjectServerError(0)
	srv.InjectServerError(-2)
	srv.InjectMetaError(100, "bad", 0)
	srv.InjectMetaError(100, "bad", -3)

	client := srv.Client()
	_, err := client.Messages.SendText(context.Background(), "1234567890", "Should succeed immediately")
	if err != nil {
		t.Fatalf("expected immediate success with zero-count injections, got: %v", err)
	}
}

func TestServer_InjectMetaError_TemplateError(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject Template error (code 132000)
	srv.InjectMetaError(132000, "Template does not exist", 1)

	client := srv.Client()
	ctx := context.Background()

	_, err := client.Messages.SendText(ctx, "1234567890", "Template test")
	if err == nil {
		t.Fatal("expected template error, got nil")
	}

	var apiErr *wacloudapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *wacloudapi.APIError, got %T: %v", err, err)
	}

	if !apiErr.IsTemplateError() {
		t.Errorf("expected IsTemplateError() to be true, got code %d", apiErr.Code)
	}
}

func TestServer_SetCustomHandler(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Register custom handler for specific path
	srv.SetCustomHandler("/v21.0/custom_endpoint", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"custom": "teapot_response",
		})
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/v21.0/custom_endpoint", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("failed to execute custom request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("expected status 418 I'm a teapot, got %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if data["custom"] != "teapot_response" {
		t.Errorf("expected %q, got %q", "teapot_response", data["custom"])
	}
}

func TestServer_SetCustomHandler_StrippedPathMatching(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Register with path without version prefix
	srv.SetCustomHandler("/stripped_endpoint", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stripped_ok"))
	})

	// Request with /v21.0/stripped_endpoint
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/v21.0/stripped_endpoint", nil)
	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "stripped_ok" {
		t.Errorf("expected stripped_ok, got %q", string(body))
	}
}

func TestServer_SetCustomHandler_MethodSpecific(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	srv.SetCustomHandler("POST /v21.0/hook", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("handled: " + string(body)))
	})

	// GET should not match the POST-only handler and should return 404 or Graph API routing
	reqGet, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/v21.0/hook", nil)
	respGet, err := srv.HTTPClient().Do(reqGet)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	_ = respGet.Body.Close()
	if respGet.StatusCode == http.StatusAccepted {
		t.Errorf("GET should not have matched POST /v21.0/hook")
	}

	// POST should match
	reqPost, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL()+"/v21.0/hook", io.NopCloser(strings.NewReader("payload123")))
	respPost, err := srv.HTTPClient().Do(reqPost)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer respPost.Body.Close()

	if respPost.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", respPost.StatusCode)
	}

	resBody, _ := io.ReadAll(respPost.Body)
	if string(resBody) != "handled: payload123" {
		t.Errorf("expected %q, got %q", "handled: payload123", string(resBody))
	}
}

func TestServer_Reset_ClearsChaosAndCustomHandlers(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Inject error and custom handler
	srv.InjectRateLimit(5)
	srv.SetCustomHandler("/v21.0/custom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Call Reset
	srv.Reset()

	// Client request should succeed now without rate limiting
	client := srv.Client()
	ctx := context.Background()
	_, err := client.Messages.SendText(ctx, "1234567890", "After reset")
	if err != nil {
		t.Fatalf("expected send text to succeed after reset, got: %v", err)
	}

	// Custom handler should be cleared
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL()+"/v21.0/custom", nil)
	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected custom handler to be removed after Reset()")
	}
}

func TestServer_Chaos_Concurrency(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%3 == 0 {
				srv.InjectRateLimit(1)
			} else if id%3 == 1 {
				srv.InjectServerError(1)
			} else {
				srv.InjectMetaError(190, "OAuth error", 1)
			}
			srv.SetCustomHandler("/concurrent", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}(i)
	}
	wg.Wait()
}
