package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTemplatesServiceCreate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-123/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body CreateTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body.Name != "seasonal_sale" {
			t.Errorf("unexpected template name: %s", body.Name)
		}
		_ = json.NewEncoder(w).Encode(CreateTemplateResponse{
			ID:       "tpl-999",
			Status:   TemplateStatusPending,
			Category: TemplateCategoryMarketing,
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))
	res, err := c.Templates.Create(context.Background(), &CreateTemplateRequest{
		Name:     "seasonal_sale",
		Category: TemplateCategoryMarketing,
		Language: "en_US",
		Components: []TemplateComponent{
			{
				Type: "BODY",
				Text: "Hello {{1}}, check out our sale!",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "tpl-999" || res.Status != TemplateStatusPending || res.Category != TemplateCategoryMarketing {
		t.Errorf("unexpected create response: %+v", res)
	}
}

func TestTemplatesServiceCreate_ExplicitWABAID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-custom/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(CreateTemplateResponse{
			ID:       "tpl-custom",
			Status:   TemplateStatusApproved,
			Category: TemplateCategoryUtility,
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	res, err := c.Templates.Create(context.Background(), &CreateTemplateRequest{
		WABAID:   "waba-custom",
		Name:     "custom_template",
		Category: TemplateCategoryUtility,
		Language: "en_US",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "tpl-custom" {
		t.Errorf("expected tpl-custom, got %s", res.ID)
	}
}

func TestTemplatesServiceCreate_Errors(t *testing.T) {
	c := New("test-token", "phone-1")

	// Nil request
	_, err := c.Templates.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}

	// Missing WABA ID
	_, err = c.Templates.Create(context.Background(), &CreateTemplateRequest{
		Name: "test_tpl",
	})
	if err == nil {
		t.Fatal("expected error for missing WABA ID")
	}
}

func TestTemplatesServiceListAndPagination(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-123/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		q := r.URL.Query()
		if q.Get("category") != "UTILITY" {
			t.Errorf("expected category UTILITY, got %s", q.Get("category"))
		}
		if q.Get("status") != "APPROVED" {
			t.Errorf("expected status APPROVED, got %s", q.Get("status"))
		}
		if q.Get("name") != "welcome_msg" {
			t.Errorf("expected name welcome_msg, got %s", q.Get("name"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("expected limit 10, got %s", q.Get("limit"))
		}
		if q.Get("after") != "cursor_1" {
			t.Errorf("expected after cursor_1, got %s", q.Get("after"))
		}
		if q.Get("before") != "cursor_0" {
			t.Errorf("expected before cursor_0, got %s", q.Get("before"))
		}

		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Data: []TemplateDetails{
				{
					ID:       "tpl-1",
					Name:     "welcome_msg",
					Status:   TemplateStatusApproved,
					Category: TemplateCategoryUtility,
					Language: "en_US",
					Components: []TemplateComponent{
						{
							Type: "BODY",
							Text: "Welcome to our store!",
						},
					},
				},
			},
			Paging: &Paging{
				Cursors:  &Cursors{Before: "cursor_0", After: "cursor_after_123"},
				Next:     "https://graph.facebook.com/v21.0/waba-123/message_templates?after=cursor_after_123",
				Previous: "https://graph.facebook.com/v21.0/waba-123/message_templates?before=cursor_0",
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))
	res, err := c.Templates.List(context.Background(), &ListTemplatesRequest{
		Category: TemplateCategoryUtility,
		Status:   TemplateStatusApproved,
		Name:     "welcome_msg",
		Limit:    10,
		After:    "cursor_1",
		Before:   "cursor_0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Data) != 1 || res.Data[0].Name != "welcome_msg" {
		t.Errorf("unexpected templates list: %+v", res)
	}
	if !res.HasNext() || res.NextCursor() != "cursor_after_123" {
		t.Errorf("expected pagination HasNext and cursor, got HasNext=%v cursor=%s", res.HasNext(), res.NextCursor())
	}
}

func TestTemplatesServiceList_NilRequestAndErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-123/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Data: []TemplateDetails{},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))
	res, err := c.Templates.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(res.Data))
	}

	// Missing WABA ID error
	cNoWABA := New("test-token", "phone-1", WithBaseURL(ts.URL))
	_, err = cNoWABA.Templates.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing WABA ID")
	}
}

func TestListTemplatesResponse_PaginationHelpers(t *testing.T) {
	var nilResp *ListTemplatesResponse
	if nilResp.HasNext() {
		t.Errorf("expected HasNext() false for nil response")
	}
	if nilResp.NextCursor() != "" {
		t.Errorf("expected NextCursor() empty for nil response")
	}
	if nilResp.HasPrevious() {
		t.Errorf("expected HasPrevious() false for nil response")
	}
	if nilResp.PreviousCursor() != "" {
		t.Errorf("expected PreviousCursor() empty for nil response")
	}

	emptyResp := &ListTemplatesResponse{}
	if emptyResp.HasNext() {
		t.Errorf("expected HasNext() false for empty response")
	}
	if emptyResp.NextCursor() != "" {
		t.Errorf("expected NextCursor() empty for empty response")
	}
	if emptyResp.HasPrevious() {
		t.Errorf("expected HasPrevious() false for empty response")
	}
	if emptyResp.PreviousCursor() != "" {
		t.Errorf("expected PreviousCursor() empty for empty response")
	}

	respWithNextURL := &ListTemplatesResponse{
		Paging: &Paging{
			Next: "https://graph.facebook.com/v21.0/waba-123/message_templates?after=abc",
		},
	}
	if !respWithNextURL.HasNext() {
		t.Errorf("expected HasNext() true for response with next URL")
	}
	if respWithNextURL.NextCursor() != "" {
		t.Errorf("expected NextCursor() empty when cursors is nil")
	}

	respWithPrevURL := &ListTemplatesResponse{
		Paging: &Paging{
			Previous: "https://graph.facebook.com/v21.0/waba-123/message_templates?before=abc",
		},
	}
	if !respWithPrevURL.HasPrevious() {
		t.Errorf("expected HasPrevious() true for response with prev URL")
	}
	if respWithPrevURL.PreviousCursor() != "" {
		t.Errorf("expected PreviousCursor() empty when cursors is nil")
	}

	respWithCursor := &ListTemplatesResponse{
		Paging: &Paging{
			Cursors: &Cursors{
				Before: "cursor_before_123",
				After:  "cursor_after_123",
			},
		},
	}
	if !respWithCursor.HasNext() {
		t.Errorf("expected HasNext() true for response with cursor")
	}
	if respWithCursor.NextCursor() != "cursor_after_123" {
		t.Errorf("expected NextCursor() cursor_after_123, got %s", respWithCursor.NextCursor())
	}
	if !respWithCursor.HasPrevious() {
		t.Errorf("expected HasPrevious() true for response with cursor")
	}
	if respWithCursor.PreviousCursor() != "cursor_before_123" {
		t.Errorf("expected PreviousCursor() cursor_before_123, got %s", respWithCursor.PreviousCursor())
	}
}

func TestTemplatesServiceNextPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") != "cursor_after_1" {
			t.Errorf("expected after cursor_after_1, got %s", r.URL.Query().Get("after"))
		}
		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Data: []TemplateDetails{
				{ID: "tpl-page2", Name: "template_2"},
			},
			Paging: &Paging{
				Cursors: &Cursors{Before: "cursor_before_2", After: "cursor_after_2"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))

	current := &ListTemplatesResponse{
		Data: []TemplateDetails{{ID: "tpl-page1"}},
		Paging: &Paging{
			Cursors: &Cursors{After: "cursor_after_1"},
		},
	}

	next, err := c.Templates.NextPage(context.Background(), current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(next.Data) != 1 || next.Data[0].ID != "tpl-page2" {
		t.Errorf("unexpected next page data: %+v", next)
	}

	// Explicit WABA ID
	nextExplicit, err := c.Templates.NextPage(context.Background(), current, "waba-123")
	if err != nil {
		t.Fatalf("unexpected error with explicit WABA: %v", err)
	}
	if len(nextExplicit.Data) != 1 || nextExplicit.Data[0].ID != "tpl-page2" {
		t.Errorf("unexpected next page data with explicit WABA: %+v", nextExplicit)
	}
}

func TestTemplatesServiceNextPage_Errors(t *testing.T) {
	c := New("test-token", "phone-1", WithWABAID("waba-123"))

	// Nil current
	_, err := c.Templates.NextPage(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil current response")
	}

	// No next page available
	noNext := &ListTemplatesResponse{}
	_, err = c.Templates.NextPage(context.Background(), noNext)
	if err == nil {
		t.Fatal("expected error when no next page available")
	}
}

func TestTemplatesServicePrevPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("before") != "cursor_before_2" {
			t.Errorf("expected before cursor_before_2, got %s", r.URL.Query().Get("before"))
		}
		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Data: []TemplateDetails{
				{ID: "tpl-page1", Name: "template_1"},
			},
			Paging: &Paging{
				Cursors: &Cursors{Before: "cursor_before_1", After: "cursor_after_1"},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))

	current := &ListTemplatesResponse{
		Data: []TemplateDetails{{ID: "tpl-page2"}},
		Paging: &Paging{
			Cursors: &Cursors{Before: "cursor_before_2"},
		},
	}

	prev, err := c.Templates.PrevPage(context.Background(), current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prev.Data) != 1 || prev.Data[0].ID != "tpl-page1" {
		t.Errorf("unexpected prev page data: %+v", prev)
	}

	// Explicit WABA ID
	prevExplicit, err := c.Templates.PrevPage(context.Background(), current, "waba-123")
	if err != nil {
		t.Fatalf("unexpected error with explicit WABA: %v", err)
	}
	if len(prevExplicit.Data) != 1 || prevExplicit.Data[0].ID != "tpl-page1" {
		t.Errorf("unexpected prev page data with explicit WABA: %+v", prevExplicit)
	}
}

func TestTemplatesServicePrevPage_Errors(t *testing.T) {
	c := New("test-token", "phone-1", WithWABAID("waba-123"))

	// Nil current
	_, err := c.Templates.PrevPage(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil current response")
	}

	// No prev page available
	noPrev := &ListTemplatesResponse{}
	_, err = c.Templates.PrevPage(context.Background(), noPrev)
	if err == nil {
		t.Fatal("expected error when no previous page available")
	}
}

func TestTemplatesServiceGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/tpl-999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(TemplateDetails{
			ID:       "tpl-999",
			Name:     "seasonal_sale",
			Status:   TemplateStatusApproved,
			Category: TemplateCategoryMarketing,
			Language: "en_US",
			Components: []TemplateComponent{
				{
					Type: "BODY",
					Text: "Hello {{1}}, check out our sale!",
				},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	details, err := c.Templates.Get(context.Background(), "tpl-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.ID != "tpl-999" || details.Name != "seasonal_sale" || details.Status != TemplateStatusApproved {
		t.Errorf("unexpected details: %+v", details)
	}

	// Empty ID error
	_, err = c.Templates.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty template ID")
	}
}

func TestTemplatesServiceUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/tpl-999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(UpdateTemplateResponse{
			Success: true,
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	res, err := c.Templates.Update(context.Background(), "tpl-999", &UpdateTemplateRequest{
		Components: []TemplateComponent{
			{
				Type: "BODY",
				Text: "Updated body text",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success true, got false")
	}

	// Empty ID error
	_, err = c.Templates.Update(context.Background(), "", &UpdateTemplateRequest{})
	if err == nil {
		t.Fatal("expected error for empty template ID")
	}

	// Nil request error
	_, err = c.Templates.Update(context.Background(), "tpl-999", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestTemplatesServiceDeleteByName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v21.0/waba-123/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "old_template" {
			t.Errorf("expected name query old_template, got %s", r.URL.Query().Get("name"))
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))
	err := c.Templates.DeleteByName(context.Background(), "old_template")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit WABA ID
	err = c.Templates.DeleteByName(context.Background(), "old_template", "waba-123")
	if err != nil {
		t.Fatalf("unexpected error with explicit WABA: %v", err)
	}

	// Empty name error
	err = c.Templates.DeleteByName(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty template name")
	}

	// Missing WABA ID error
	cNoWABA := New("test-token", "phone-1", WithBaseURL(ts.URL))
	err = cNoWABA.Templates.DeleteByName(context.Background(), "old_template")
	if err == nil {
		t.Fatal("expected error for missing WABA ID")
	}
}

func TestTemplatesServiceDeleteByID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v21.0/tpl-999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithBaseURL(ts.URL))
	err := c.Templates.DeleteByID(context.Background(), "tpl-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty ID error
	err = c.Templates.DeleteByID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty template ID")
	}
}
