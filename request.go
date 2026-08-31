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
		if ctx.Err() != nil {
			return ctx.Err()
		}

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

		// Check if retryable (RateLimit or 5xx)
		if (apiErr.IsRateLimit() || resp.StatusCode >= 500) && attempt < maxAttempts {
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
