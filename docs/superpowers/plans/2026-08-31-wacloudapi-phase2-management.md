# WhatsApp Business Management APIs (Phase 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun WhatsApp Business Management APIs pada package Go `github.com/itsmeabde/wacloudapi` mencakup Business Profile, Message Templates, Phone Numbers & 2FA, serta Message QR Codes.

**Architecture:** Menambahkan sub-services baru pada `*Client` (`client.BusinessProfile`, `client.Templates`, `client.PhoneNumbers`, `client.QRCodes`), opsi `WithWABAID(wabaID)`, cursor pagination helpers untuk templates, serta image stream downloader untuk QR Codes.

**Tech Stack:** Go 1.22+ (Zero external dependencies, 100% Go Standard Library: `net/http`, `encoding/json`, `io`, `time`, `context`, `fmt`, `net/url`).

## Global Constraints
- Go Version: `go 1.22+`
- External Dependencies: 0 (Zero external dependencies, only Go standard library)
- Module Name: `github.com/itsmeabde/wacloudapi`
- Default API Version: `v21.0`
- Default Base URL: `https://graph.facebook.com`
- Concurrency: All exported client and handler methods must be safe for concurrent goroutine access
- Context: All API network calls must accept and respect `context.Context`

---

### Task 1: WABAID Config Option & Business Profile Service

**Files:**
- Modify: `config.go` (add `WABAID` field)
- Modify: `options.go` (add `WithWABAID(string)` option)
- Create: `profile_models.go`
- Create: `profile.go`
- Modify: `client.go` (attach `client.BusinessProfile`)
- Test: `profile_test.go`

**Interfaces:**
- Produces:
  - `WithWABAID(wabaID string) Option`
  - `BusinessProfile`, `UpdateBusinessProfileRequest`, `Vertical`
  - `(s *BusinessProfileService) Get(ctx context.Context, fields ...string) (*BusinessProfile, error)`
  - `(s *BusinessProfileService) Update(ctx context.Context, req *UpdateBusinessProfileRequest) error`

- [ ] **Step 1: Write failing tests for BusinessProfileService and WithWABAID**

Create `profile_test.go`:
```go
package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithWABAIDOption(t *testing.T) {
	c := New("test-token", "12345", WithWABAID("waba-999"))
	if c.config.WABAID != "waba-999" {
		t.Errorf("expected WABAID waba-999, got %s", c.config.WABAID)
	}
}

func TestBusinessProfileGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/12345/whatsapp_business_profile" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("fields") != "about,address,description,email,profile_picture_url,websites,vertical" {
			t.Errorf("unexpected fields query: %s", r.URL.Query().Get("fields"))
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []BusinessProfile{
				{
					About:             "About company",
					Address:           "Jakarta, Indonesia",
					Description:       "Official Store",
					Email:             "contact@company.com",
					ProfilePictureURL: "https://lookaside.fbsbx.com/photo.jpg",
					Websites:          []string{"https://company.com"},
					Vertical:          VerticalRetail,
					MessagingProduct:  "whatsapp",
				},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	profile, err := c.BusinessProfile.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.About != "About company" || profile.Vertical != VerticalRetail {
		t.Errorf("unexpected profile: %+v", profile)
	}
}

func TestBusinessProfileUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		var req UpdateBusinessProfileRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.About != "New about" || req.Vertical != VerticalProfServices {
			t.Errorf("unexpected update body: %+v", req)
		}

		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	err := c.BusinessProfile.Update(context.Background(), &UpdateBusinessProfileRequest{
		About:    "New about",
		Vertical: VerticalProfServices,
		Websites: []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestBusinessProfile ./...`
Expected: FAIL.

- [ ] **Step 3: Implement WABAID option and BusinessProfileService**

Update `config.go` to include `WABAID string` in `Config`.
Update `options.go` to include:
```go
func WithWABAID(wabaID string) Option {
	return func(c *Config) {
		if wabaID != "" {
			c.WABAID = wabaID
		}
	}
}
```

Create `profile_models.go`:
```go
package wacloudapi

type Vertical string

const (
	VerticalUndefined    Vertical = "UNDEFINED"
	VerticalOther        Vertical = "OTHER"
	VerticalAuto         Vertical = "AUTO"
	VerticalBeauty       Vertical = "BEAUTY"
	VerticalApparel      Vertical = "APPAREL"
	VerticalEdu          Vertical = "EDU"
	VerticalEntertain    Vertical = "ENTERTAIN"
	VerticalEventPlan    Vertical = "EVENT_PLAN"
	VerticalFinance      Vertical = "FINANCE"
	VerticalGrocery      Vertical = "GROCERY"
	VerticalGovt         Vertical = "GOVT"
	VerticalHotel        Vertical = "HOTEL"
	VerticalHealth       Vertical = "HEALTH"
	VerticalNonprofit    Vertical = "NONPROFIT"
	VerticalProfServices Vertical = "PROF_SERVICES"
	VerticalRetail       Vertical = "RETAIL"
	VerticalTravel       Vertical = "TRAVEL"
	VerticalRestaurant   Vertical = "RESTAURANT"
)

type BusinessProfile struct {
	About             string   `json:"about,omitempty"`
	Address           string   `json:"address,omitempty"`
	Description       string   `json:"description,omitempty"`
	Email             string   `json:"email,omitempty"`
	ProfilePictureURL string   `json:"profile_picture_url,omitempty"`
	Websites          []string `json:"websites,omitempty"`
	Vertical          Vertical `json:"vertical,omitempty"`
	MessagingProduct  string   `json:"messaging_product,omitempty"`
}

type UpdateBusinessProfileRequest struct {
	MessagingProduct string   `json:"messaging_product"`
	About            string   `json:"about,omitempty"`
	Address          string   `json:"address,omitempty"`
	Description      string   `json:"description,omitempty"`
	Email            string   `json:"email,omitempty"`
	Websites         []string `json:"websites,omitempty"`
	Vertical         Vertical `json:"vertical,omitempty"`
}
```

Create `profile.go`:
```go
package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type BusinessProfileService struct {
	client *Client
}

func newBusinessProfileService(client *Client) *BusinessProfileService {
	return &BusinessProfileService{client: client}
}

func (s *BusinessProfileService) Get(ctx context.Context, fields ...string) (*BusinessProfile, error) {
	fieldQuery := "about,address,description,email,profile_picture_url,websites,vertical"
	if len(fields) > 0 && strings.TrimSpace(fields[0]) != "" {
		fieldQuery = strings.Join(fields, ",")
	}

	endpoint := fmt.Sprintf("%s/whatsapp_business_profile?fields=%s",
		s.client.config.PhoneNumberID,
		url.QueryEscape(fieldQuery),
	)

	var wrapper struct {
		Data []BusinessProfile `json:"data"`
	}
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &wrapper); err != nil {
		return nil, err
	}

	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("wacloudapi: no business profile data found")
	}

	return &wrapper.Data[0], nil
}

func (s *BusinessProfileService) Update(ctx context.Context, req *UpdateBusinessProfileRequest) error {
	if req == nil {
		return fmt.Errorf("wacloudapi: update request cannot be nil")
	}
	req.MessagingProduct = "whatsapp"

	endpoint := fmt.Sprintf("%s/whatsapp_business_profile", s.client.config.PhoneNumberID)
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, req, nil)
}
```

Update `client.go` to attach `c.BusinessProfile = newBusinessProfileService(c)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestBusinessProfile ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config.go options.go profile_models.go profile.go client.go profile_test.go
git commit -m "feat: implement BusinessProfileService and WithWABAID configuration option"
```

---

### Task 2: Message Template Management Service

**Files:**
- Create: `templates_models.go`
- Create: `templates.go`
- Modify: `client.go` (attach `client.Templates`)
- Test: `templates_test.go`

**Interfaces:**
- Produces:
  - Models: `CreateTemplateRequest`, `CreateTemplateResponse`, `ListTemplatesRequest`, `ListTemplatesResponse`, `TemplateDetails`, `UpdateTemplateRequest`, `UpdateTemplateResponse`, `TemplateCategory`, `TemplateStatus`
  - Methods on `TemplatesService`:
    - `Create(ctx context.Context, req *CreateTemplateRequest) (*CreateTemplateResponse, error)`
    - `List(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error)`
    - `Get(ctx context.Context, templateID string) (*TemplateDetails, error)`
    - `Update(ctx context.Context, templateID string, req *UpdateTemplateRequest) (*UpdateTemplateResponse, error)`
    - `DeleteByName(ctx context.Context, name string, wabaID ...string) error`
    - `DeleteByID(ctx context.Context, templateID string) error`

- [ ] **Step 1: Write failing tests for TemplatesService (Create, List with Pagination, Get, Update, Delete)**

Create `templates_test.go`:
```go
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
	if res.ID != "tpl-999" || res.Status != TemplateStatusPending {
		t.Errorf("unexpected create response: %+v", res)
	}
}

func TestTemplatesServiceListAndPagination(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-123/message_templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Data: []TemplateDetails{
				{ID: "tpl-1", Name: "welcome_msg", Status: TemplateStatusApproved},
			},
			Paging: &Paging{
				Cursors: &Cursors{After: "cursor_after_123"},
				Next:    "https://graph.facebook.com/v21.0/waba-123/message_templates?after=cursor_after_123",
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "phone-1", WithWABAID("waba-123"), WithBaseURL(ts.URL))
	res, err := c.Templates.List(context.Background(), &ListTemplatesRequest{
		Category: TemplateCategoryUtility,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Data) != 1 || res.Data[0].Name != "welcome_msg" {
		t.Errorf("unexpected templates list: %+v", res)
	}
	if !res.HasNext() || res.NextCursor() != "cursor_after_123" {
		t.Errorf("expected pagination HasNext and cursor")
	}
}

func TestTemplatesServiceDeleteByName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
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
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestTemplatesService ./...`
Expected: FAIL.

- [ ] **Step 3: Implement template management models and TemplatesService**

Create `templates_models.go`:
```go
package wacloudapi

type TemplateCategory string

const (
	TemplateCategoryMarketing      TemplateCategory = "MARKETING"
	TemplateCategoryUtility        TemplateCategory = "UTILITY"
	TemplateCategoryAuthentication TemplateCategory = "AUTHENTICATION"
)

type TemplateStatus string

const (
	TemplateStatusApproved TemplateStatus = "APPROVED"
	TemplateStatusPending  TemplateStatus = "PENDING"
	TemplateStatusRejected TemplateStatus = "REJECTED"
	TemplateStatusPaused   TemplateStatus = "PAUSED"
	TemplateStatusDisabled TemplateStatus = "DISABLED"
)

type CreateTemplateRequest struct {
	WABAID     string              `json:"-"`
	Name       string              `json:"name"`
	Category   TemplateCategory    `json:"category"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components"`
}

type CreateTemplateResponse struct {
	ID       string           `json:"id"`
	Status   TemplateStatus   `json:"status"`
	Category TemplateCategory `json:"category"`
}

type ListTemplatesRequest struct {
	WABAID   string           `json:"-"`
	Category TemplateCategory `json:"category,omitempty"`
	Status   TemplateStatus   `json:"status,omitempty"`
	Name     string           `json:"name,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	After    string           `json:"after,omitempty"`
	Before   string           `json:"before,omitempty"`
}

type Cursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type Paging struct {
	Cursors  *Cursors `json:"cursors,omitempty"`
	Next     string   `json:"next,omitempty"`
	Previous string   `json:"previous,omitempty"`
}

type ListTemplatesResponse struct {
	Data   []TemplateDetails `json:"data"`
	Paging *Paging           `json:"paging,omitempty"`
}

func (r *ListTemplatesResponse) HasNext() bool {
	return r != nil && r.Paging != nil && (r.Paging.Next != "" || (r.Paging.Cursors != nil && r.Paging.Cursors.After != ""))
}

func (r *ListTemplatesResponse) NextCursor() string {
	if r != nil && r.Paging != nil && r.Paging.Cursors != nil {
		return r.Paging.Cursors.After
	}
	return ""
}

type TemplateDetails struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Status     TemplateStatus      `json:"status"`
	Category   TemplateCategory    `json:"category"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components"`
}

type UpdateTemplateRequest struct {
	Components []TemplateComponent `json:"components"`
}

type UpdateTemplateResponse struct {
	Success bool `json:"success"`
}
```

Create `templates.go`:
```go
package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type TemplatesService struct {
	client *Client
}

func newTemplatesService(client *Client) *TemplatesService {
	return &TemplatesService{client: client}
}

func (s *TemplatesService) resolveWABAID(explicitID string) (string, error) {
	if explicitID != "" {
		return explicitID, nil
	}
	if s.client.config.WABAID != "" {
		return s.client.config.WABAID, nil
	}
	return "", fmt.Errorf("wacloudapi: WABA ID is required for template operations (use WithWABAID() option or specify in request)")
}

func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*CreateTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: request cannot be nil")
	}

	wabaID, err := s.resolveWABAID(req.WABAID)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/message_templates", wabaID)
	var resp CreateTemplateResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) List(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	var explicitWABA string
	params := url.Values{}
	if req != nil {
		explicitWABA = req.WABAID
		if req.Category != "" {
			params.Set("category", string(req.Category))
		}
		if req.Status != "" {
			params.Set("status", string(req.Status))
		}
		if req.Name != "" {
			params.Set("name", req.Name)
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.Itoa(req.Limit))
		}
		if req.After != "" {
			params.Set("after", req.After)
		}
		if req.Before != "" {
			params.Set("before", req.Before)
		}
	}

	wabaID, err := s.resolveWABAID(explicitWABA)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/message_templates", wabaID)
	if q := params.Encode(); q != "" {
		endpoint += "?" + q
	}

	var resp ListTemplatesResponse
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) Get(ctx context.Context, templateID string) (*TemplateDetails, error) {
	if templateID == "" {
		return nil, fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	endpoint := templateID
	var details TemplateDetails
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

func (s *TemplatesService) Update(ctx context.Context, templateID string, req *UpdateTemplateRequest) (*UpdateTemplateResponse, error) {
	if templateID == "" {
		return nil, fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: update request cannot be nil")
	}

	endpoint := templateID
	var resp UpdateTemplateResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) DeleteByName(ctx context.Context, name string, wabaID ...string) error {
	if name == "" {
		return fmt.Errorf("wacloudapi: template name cannot be empty")
	}
	var explicitWABA string
	if len(wabaID) > 0 {
		explicitWABA = wabaID[0]
	}
	id, err := s.resolveWABAID(explicitWABA)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/message_templates?name=%s", id, url.QueryEscape(name))
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (s *TemplatesService) DeleteByID(ctx context.Context, templateID string) error {
	if templateID == "" {
		return fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	endpoint := templateID
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}
```

Update `client.go` to attach `c.Templates = newTemplatesService(c)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestTemplatesService ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add templates_models.go templates.go client.go templates_test.go
git commit -m "feat: implement TemplatesService for WhatsApp Message Template management"
```

---

### Task 3: Phone Numbers & Two-Step Verification Service

**Files:**
- Create: `phone_numbers_models.go`
- Create: `phone_numbers.go`
- Modify: `client.go` (attach `client.PhoneNumbers`)
- Test: `phone_numbers_test.go`

**Interfaces:**
- Produces:
  - Models: `PhoneNumberDetails`, `ListPhoneNumbersResponse`, `CodeMethod`
  - Methods on `PhoneNumbersService`:
    - `List(ctx context.Context, wabaID ...string) (*ListPhoneNumbersResponse, error)`
    - `Get(ctx context.Context, phoneNumberID string) (*PhoneNumberDetails, error)`
    - `RequestCode(ctx context.Context, phoneNumberID string, method CodeMethod, language string) error`
    - `VerifyCode(ctx context.Context, phoneNumberID string, code string) error`
    - `Register(ctx context.Context, phoneNumberID, pin string, cert ...string) error`
    - `Deregister(ctx context.Context, phoneNumberID string) error`
    - `SetTwoStepPIN(ctx context.Context, phoneNumberID, pin string) error`

- [ ] **Step 1: Write failing tests for PhoneNumbersService**

Create `phone_numbers_test.go`:
```go
package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhoneNumbersList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-999/phone_numbers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListPhoneNumbersResponse{
			Data: []PhoneNumberDetails{
				{
					ID:                 "123456",
					DisplayPhoneNumber: "+62 812-3456-789",
					VerifiedName:       "Official Business",
					QualityRating:      "GREEN",
				},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithWABAID("waba-999"), WithBaseURL(ts.URL))
	res, err := c.PhoneNumbers.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Data) != 1 || res.Data[0].VerifiedName != "Official Business" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestPhoneNumbersRegisterAndPIN(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/123456/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["messaging_product"] != "whatsapp" || body["pin"] != "123456" {
			t.Errorf("unexpected register body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.Register(context.Background(), "123456", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestPhoneNumbers ./...`
Expected: FAIL.

- [ ] **Step 3: Implement phone numbers models and PhoneNumbersService**

Create `phone_numbers_models.go`:
```go
package wacloudapi

type CodeMethod string

const (
	CodeMethodSMS   CodeMethod = "SMS"
	CodeMethodVoice CodeMethod = "VOICE"
)

type PhoneNumberDetails struct {
	ID                      string `json:"id"`
	DisplayPhoneNumber      string `json:"display_phone_number"`
	DisplayPhoneNumberClean string `json:"display_phone_number_clean,omitempty"`
	VerifiedName            string `json:"verified_name"`
	QualityRating           string `json:"quality_rating"`
	CodeVerificationStatus  string `json:"code_verification_status,omitempty"`
	EligibilityForAPIStatus string `json:"eligibility_for_api_business_verification,omitempty"`
	NameStatus              string `json:"name_status,omitempty"`
	NewNameStatus           string `json:"new_name_status,omitempty"`
	Status                  string `json:"status,omitempty"`
}

type ListPhoneNumbersResponse struct {
	Data   []PhoneNumberDetails `json:"data"`
	Paging *Paging              `json:"paging,omitempty"`
}
```

Create `phone_numbers.go`:
```go
package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
)

type PhoneNumbersService struct {
	client *Client
}

func newPhoneNumbersService(client *Client) *PhoneNumbersService {
	return &PhoneNumbersService{client: client}
}

func (s *PhoneNumbersService) resolveWABAID(explicitID string) (string, error) {
	if explicitID != "" {
		return explicitID, nil
	}
	if s.client.config.WABAID != "" {
		return s.client.config.WABAID, nil
	}
	return "", fmt.Errorf("wacloudapi: WABA ID is required to list phone numbers")
}

func (s *PhoneNumbersService) List(ctx context.Context, wabaID ...string) (*ListPhoneNumbersResponse, error) {
	var explicitWABA string
	if len(wabaID) > 0 {
		explicitWABA = wabaID[0]
	}
	id, err := s.resolveWABAID(explicitWABA)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/phone_numbers", id)
	var resp ListPhoneNumbersResponse
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *PhoneNumbersService) Get(ctx context.Context, phoneNumberID string) (*PhoneNumberDetails, error) {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if phoneNumberID == "" {
		return nil, fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
	}

	endpoint := phoneNumberID
	var resp PhoneNumberDetails
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *PhoneNumbersService) RequestCode(ctx context.Context, phoneNumberID string, method CodeMethod, language string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if language == "" {
		language = "en_US"
	}
	if method == "" {
		method = CodeMethodSMS
	}

	endpoint := fmt.Sprintf("%s/request_code", phoneNumberID)
	body := map[string]string{
		"code_method": string(method),
		"language":    language,
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}

func (s *PhoneNumbersService) VerifyCode(ctx context.Context, phoneNumberID string, code string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if code == "" {
		return fmt.Errorf("wacloudapi: verification code cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/verify_code", phoneNumberID)
	body := map[string]string{
		"code": code,
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}

func (s *PhoneNumbersService) Register(ctx context.Context, phoneNumberID, pin string, cert ...string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if pin == "" {
		return fmt.Errorf("wacloudapi: 2FA pin cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/register", phoneNumberID)
	body := map[string]string{
		"messaging_product": "whatsapp",
		"pin":               pin,
	}
	if len(cert) > 0 && cert[0] != "" {
		body["cert"] = cert[0]
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}

func (s *PhoneNumbersService) Deregister(ctx context.Context, phoneNumberID string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	endpoint := fmt.Sprintf("%s/deregister", phoneNumberID)
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, nil, nil)
}

func (s *PhoneNumbersService) SetTwoStepPIN(ctx context.Context, phoneNumberID, pin string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if pin == "" {
		return fmt.Errorf("wacloudapi: 2FA pin cannot be empty")
	}
	endpoint := phoneNumberID
	body := map[string]string{
		"pin": pin,
	}
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, body, nil)
}
```

Update `client.go` to attach `c.PhoneNumbers = newPhoneNumbersService(c)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestPhoneNumbers ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add phone_numbers_models.go phone_numbers.go client.go phone_numbers_test.go
git commit -m "feat: implement PhoneNumbersService for phone registration and 2FA PIN management"
```

---

### Task 4: Message QR Codes Management Service

**Files:**
- Create: `qr_codes_models.go`
- Create: `qr_codes.go`
- Modify: `client.go` (attach `client.QRCodes`)
- Test: `qr_codes_test.go`

**Interfaces:**
- Produces:
  - Models: `QRCodeImageFormat`, `CreateQRCodeRequest`, `QRCodeDetails`, `ListQRCodesResponse`
  - Methods on `QRCodesService`:
    - `Create(ctx context.Context, req *CreateQRCodeRequest) (*QRCodeDetails, error)`
    - `List(ctx context.Context) (*ListQRCodesResponse, error)`
    - `Get(ctx context.Context, qrCodeID string) (*QRCodeDetails, error)`
    - `Update(ctx context.Context, qrCodeID, prefilledMessage string) (*QRCodeDetails, error)`
    - `Delete(ctx context.Context, qrCodeID string) error`
    - `DownloadImage(ctx context.Context, qrCodeID string) ([]byte, error)`

- [ ] **Step 1: Write failing tests for QRCodesService**

Create `qr_codes_test.go`:
```go
package wacloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQRCodesServiceCreateAndDownload(t *testing.T) {
	var imageServer *httptest.Server
	imageServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<svg>QR Code</svg>"))
	}))
	defer imageServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v21.0/phone-1/message_qrdls" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(QRCodeDetails{
				Code:             "QR_CODE_123",
				PrefilledMessage: "Hello from QR",
				DeepLinkURL:      "https://wa.me/message/QR_CODE_123",
				QRImageURL:       imageServer.URL,
			})
			return
		}
		if r.URL.Path == "/v21.0/phone-1/message_qrdls/QR_CODE_123" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(struct {
				Data []QRCodeDetails `json:"data"`
			}{
				Data: []QRCodeDetails{
					{
						Code:             "QR_CODE_123",
						PrefilledMessage: "Hello from QR",
						QRImageURL:       imageServer.URL,
					},
				},
			})
			return
		}
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))
	qr, err := c.QRCodes.Create(context.Background(), &CreateQRCodeRequest{
		PrefilledMessage: "Hello from QR",
		ImageFormat:      QRCodeImageSVG,
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if qr.Code != "QR_CODE_123" {
		t.Errorf("unexpected code: %s", qr.Code)
	}

	imgBytes, err := c.QRCodes.DownloadImage(context.Background(), "QR_CODE_123")
	if err != nil {
		t.Fatalf("unexpected download error: %v", err)
	}
	if string(imgBytes) != "<svg>QR Code</svg>" {
		t.Errorf("unexpected image bytes: %s", string(imgBytes))
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v -run TestQRCodes ./...`
Expected: FAIL.

- [ ] **Step 3: Implement QR Codes models and QRCodesService**

Create `qr_codes_models.go`:
```go
package wacloudapi

type QRCodeImageFormat string

const (
	QRCodeImageSVG QRCodeImageFormat = "SVG"
	QRCodeImagePNG QRCodeImageFormat = "PNG"
)

type CreateQRCodeRequest struct {
	PrefilledMessage string            `json:"prefilled_message"`
	ImageFormat      QRCodeImageFormat `json:"generate_qr_image,omitempty"`
}

type QRCodeDetails struct {
	Code             string `json:"code"`
	PrefilledMessage string `json:"prefilled_message"`
	DeepLinkURL      string `json:"deep_link_url"`
	QRImageURL       string `json:"qr_image_url"`
}

type ListQRCodesResponse struct {
	Data []QRCodeDetails `json:"data"`
}
```

Create `qr_codes.go`:
```go
package wacloudapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type QRCodesService struct {
	client *Client
}

func newQRCodesService(client *Client) *QRCodesService {
	return &QRCodesService{client: client}
}

func (s *QRCodesService) Create(ctx context.Context, req *CreateQRCodeRequest) (*QRCodeDetails, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: create qr code request cannot be nil")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls", s.client.config.PhoneNumberID)
	var resp QRCodeDetails
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) List(ctx context.Context) (*ListQRCodesResponse, error) {
	endpoint := fmt.Sprintf("%s/message_qrdls", s.client.config.PhoneNumberID)
	var resp ListQRCodesResponse
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) Get(ctx context.Context, qrCodeID string) (*QRCodeDetails, error) {
	if qrCodeID == "" {
		return nil, fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	var wrapper struct {
		Data []QRCodeDetails `json:"data"`
	}
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("wacloudapi: qr code not found")
	}
	return &wrapper.Data[0], nil
}

func (s *QRCodesService) Update(ctx context.Context, qrCodeID, prefilledMessage string) (*QRCodeDetails, error) {
	if qrCodeID == "" {
		return nil, fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	body := map[string]string{
		"prefilled_message": prefilledMessage,
	}
	var resp QRCodeDetails
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) Delete(ctx context.Context, qrCodeID string) error {
	if qrCodeID == "" {
		return fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}
	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (s *QRCodesService) DownloadImage(ctx context.Context, qrCodeID string) ([]byte, error) {
	details, err := s.Get(ctx, qrCodeID)
	if err != nil {
		return nil, err
	}
	if details.QRImageURL == "" {
		return nil, fmt.Errorf("wacloudapi: qr_image_url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, details.QRImageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to create download request: %w", err)
	}

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to download QR image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wacloudapi: download QR image failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
```

Update `client.go` to attach `c.QRCodes = newQRCodesService(c)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestQRCodes ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add qr_codes_models.go qr_codes.go client.go qr_codes_test.go
git commit -m "feat: implement QRCodesService for WhatsApp Message QR code management"
```

---

### Task 5: New Examples, Documentation & Full Verification

**Files:**
- Create: `examples/business_profile/main.go`
- Create: `examples/templates_management/main.go`
- Create: `examples/qr_codes/main.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: All Phase 1 and Phase 2 services

- [ ] **Step 1: Create runnable examples for Phase 2**

Create `examples/business_profile/main.go`:
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

	if token == "" || phoneID == "" {
		log.Fatal("WA_ACCESS_TOKEN and WA_PHONE_NUMBER_ID must be set")
	}

	client := wacloudapi.New(token, phoneID)

	// Get Profile
	profile, err := client.BusinessProfile.Get(context.Background())
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}
	fmt.Printf("Profile About: %s, Vertical: %s\n", profile.About, profile.Vertical)

	// Update Profile
	err = client.BusinessProfile.Update(context.Background(), &wacloudapi.UpdateBusinessProfileRequest{
		About:       "Official Customer Support",
		Description: "Contact us for inquiries and help.",
		Vertical:    wacloudapi.VerticalRetail,
		Websites:    []string{"https://example.com"},
	})
	if err != nil {
		log.Fatalf("Failed to update profile: %v", err)
	}
	fmt.Println("Profile successfully updated!")
}
```

Create `examples/templates_management/main.go`:
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
	wabaID := os.Getenv("WA_WABA_ID")

	if token == "" || phoneID == "" || wabaID == "" {
		log.Fatal("WA_ACCESS_TOKEN, WA_PHONE_NUMBER_ID, and WA_WABA_ID must be set")
	}

	client := wacloudapi.New(token, phoneID, wacloudapi.WithWABAID(wabaID))

	// List Templates
	list, err := client.Templates.List(context.Background(), &wacloudapi.ListTemplatesRequest{
		Limit: 5,
	})
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("Found %d templates:\n", len(list.Data))
	for _, tpl := range list.Data {
		fmt.Printf("- %s [%s] (%s)\n", tpl.Name, tpl.Category, tpl.Status)
	}
}
```

Create `examples/qr_codes/main.go`:
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

	if token == "" || phoneID == "" {
		log.Fatal("WA_ACCESS_TOKEN and WA_PHONE_NUMBER_ID must be set")
	}

	client := wacloudapi.New(token, phoneID)

	// Create QR Code
	qr, err := client.QRCodes.Create(context.Background(), &wacloudapi.CreateQRCodeRequest{
		PrefilledMessage: "Halo, saya ingin bertanya tentang produk.",
		ImageFormat:      wacloudapi.QRCodeImagePNG,
	})
	if err != nil {
		log.Fatalf("Failed to create QR code: %v", err)
	}
	fmt.Printf("QR Code created! Code: %s, DeepLink: %s\n", qr.Code, qr.DeepLinkURL)

	// List QR Codes
	list, err := client.QRCodes.List(context.Background())
	if err != nil {
		log.Fatalf("Failed to list QR codes: %v", err)
	}
	fmt.Printf("Total QR Codes: %d\n", len(list.Data))
}
```

Update `README.md` to document Phase 2 Management APIs (`BusinessProfile`, `Templates`, `PhoneNumbers`, `QRCodes`).

- [ ] **Step 2: Run all tests with -race and verify example compilation**

Run: `GOWORK=off go test -v -race -cover ./...`
Run: `GOWORK=off go vet ./...`
Run: `GOWORK=off go build ./examples/...`
Expected: ALL PASS.

- [ ] **Step 3: Commit**

```bash
git add examples/ README.md
git commit -m "docs: add Phase 2 examples, update README with Management APIs, and verify full test suite"
```
