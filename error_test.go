package wacloudapi

import (
	"strings"
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

func TestAPIErrorAdditionalPredicates(t *testing.T) {
	// Test other rate limit codes: 130429, 613, HTTP 429
	tests := []struct {
		err         *APIError
		isRateLimit bool
		isAuth      bool
		isTemplate  bool
	}{
		{
			err:         &APIError{Code: 130429},
			isRateLimit: true,
		},
		{
			err:         &APIError{Code: 613},
			isRateLimit: true,
		},
		{
			err:         &APIError{HTTPStatusCode: 429},
			isRateLimit: true,
		},
		{
			err:    &APIError{Code: 102},
			isAuth: true,
		},
		{
			err:    &APIError{HTTPStatusCode: 401},
			isAuth: true,
		},
		{
			err:        &APIError{Code: 132999},
			isTemplate: true,
		},
		{
			err:        &APIError{Code: 131000},
			isTemplate: false,
		},
		{
			err:         nil,
			isRateLimit: false,
			isAuth:      false,
			isTemplate:  false,
		},
	}

	for i, tc := range tests {
		if tc.err.IsRateLimit() != tc.isRateLimit {
			t.Errorf("[%d] expected IsRateLimit=%v, got %v", i, tc.isRateLimit, tc.err.IsRateLimit())
		}
		if tc.err.IsAuthError() != tc.isAuth {
			t.Errorf("[%d] expected IsAuthError=%v, got %v", i, tc.isAuth, tc.err.IsAuthError())
		}
		if tc.err.IsTemplateError() != tc.isTemplate {
			t.Errorf("[%d] expected IsTemplateError=%v, got %v", i, tc.isTemplate, tc.err.IsTemplateError())
		}
	}
}

func TestAPIErrorFormatting(t *testing.T) {
	var nilErr *APIError
	if nilErr.Error() != "<nil>" {
		t.Errorf("expected '<nil>', got %s", nilErr.Error())
	}

	errWithUserMsg := &APIError{
		HTTPStatusCode: 400,
		Code:           100,
		ErrorSubcode:   33,
		Message:        "Param error",
		ErrorUserTitle: "Invalid Input",
		ErrorUserMsg:   "The input parameter was invalid.",
	}
	formatted := errWithUserMsg.Error()
	if !strings.Contains(formatted, "Invalid Input") || !strings.Contains(formatted, "The input parameter was invalid.") {
		t.Errorf("expected user title and msg in error string, got %s", formatted)
	}

	errWithTrace := &APIError{
		HTTPStatusCode: 500,
		Code:           2,
		ErrorSubcode:   0,
		Message:        "Service error",
		FBTraceID:      "trace_12345",
	}
	formattedTrace := errWithTrace.Error()
	if !strings.Contains(formattedTrace, "trace_12345") {
		t.Errorf("expected trace id in error string, got %s", formattedTrace)
	}
}
