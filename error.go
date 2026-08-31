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
