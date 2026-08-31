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
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
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

func TestPhoneNumbersList_ExplicitWABAID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/waba-custom/phone_numbers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ListPhoneNumbersResponse{
			Data: []PhoneNumberDetails{
				{
					ID:                 "999999",
					DisplayPhoneNumber: "+1 555-0100",
					VerifiedName:       "Custom Business",
					QualityRating:      "GREEN",
				},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	res, err := c.PhoneNumbers.List(context.Background(), "waba-custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Data) != 1 || res.Data[0].ID != "999999" {
		t.Errorf("unexpected response: %+v", res)
	}
}

func TestPhoneNumbersList_MissingWABAID(t *testing.T) {
	c := New("test-token", "123456")
	_, err := c.PhoneNumbers.List(context.Background())
	if err == nil {
		t.Fatal("expected error for missing WABA ID")
	}
}

func TestPhoneNumbersGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-explicit" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(PhoneNumberDetails{
			ID:                      "phone-explicit",
			DisplayPhoneNumber:      "+1 555-0199",
			DisplayPhoneNumberClean: "15550199",
			VerifiedName:            "Acme Corp",
			QualityRating:           "GREEN",
			CodeVerificationStatus:  "VERIFIED",
			EligibilityForAPIStatus: "ELIGIBLE",
			NameStatus:              "APPROVED",
			Status:                  "CONNECTED",
		})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	details, err := c.PhoneNumbers.Get(context.Background(), "phone-explicit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.ID != "phone-explicit" || details.VerifiedName != "Acme Corp" || details.QualityRating != "GREEN" {
		t.Errorf("unexpected phone details: %+v", details)
	}
}

func TestPhoneNumbersGet_FallbackClientID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/default-phone-id" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(PhoneNumberDetails{
			ID:           "default-phone-id",
			VerifiedName: "Default Phone",
		})
	}))
	defer ts.Close()

	c := New("test-token", "default-phone-id", WithBaseURL(ts.URL))
	details, err := c.PhoneNumbers.Get(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details.ID != "default-phone-id" {
		t.Errorf("expected default-phone-id, got %s", details.ID)
	}
}

func TestPhoneNumbersGet_EmptyIDError(t *testing.T) {
	c := New("test-token", "")
	_, err := c.PhoneNumbers.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}

func TestPhoneNumbersRequestCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/request_code" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code_method"] != "VOICE" || body["language"] != "id_ID" {
			t.Errorf("unexpected request_code body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.RequestCode(context.Background(), "phone-1", CodeMethodVoice, "id_ID")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPhoneNumbersRequestCode_DefaultsAndFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/fallback-phone/request_code" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code_method"] != "SMS" || body["language"] != "en_US" {
			t.Errorf("unexpected request_code default body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "fallback-phone", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.RequestCode(context.Background(), "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPhoneNumbersRequestCode_EmptyIDError(t *testing.T) {
	c := New("test-token", "")
	err := c.PhoneNumbers.RequestCode(context.Background(), "", CodeMethodSMS, "en_US")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}

func TestPhoneNumbersVerifyCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/verify_code" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code"] != "123456" {
			t.Errorf("unexpected verify_code body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.VerifyCode(context.Background(), "phone-1", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPhoneNumbersVerifyCode_Errors(t *testing.T) {
	c := New("test-token", "phone-1")

	// Empty code error
	err := c.PhoneNumbers.VerifyCode(context.Background(), "phone-1", "")
	if err == nil {
		t.Fatal("expected error for empty code")
	}

	// Empty phone ID error
	cEmptyPhone := New("test-token", "")
	err = cEmptyPhone.PhoneNumbers.VerifyCode(context.Background(), "", "123456")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}

func TestPhoneNumbersRegisterAndPIN(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/123456/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
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

func TestPhoneNumbersRegister_WithCert(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/123456/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["messaging_product"] != "whatsapp" || body["pin"] != "123456" || body["cert"] != "cert-blob-123" {
			t.Errorf("unexpected register body with cert: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.Register(context.Background(), "123456", "123456", "cert-blob-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPhoneNumbersRegister_Errors(t *testing.T) {
	c := New("test-token", "123456")

	// Empty PIN error
	err := c.PhoneNumbers.Register(context.Background(), "123456", "")
	if err == nil {
		t.Fatal("expected error for empty PIN")
	}

	// Empty phone ID error
	cEmpty := New("test-token", "")
	err = cEmpty.PhoneNumbers.Register(context.Background(), "", "123456")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}

func TestPhoneNumbersDeregister(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/deregister" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.Deregister(context.Background(), "phone-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty phone ID error
	cEmpty := New("test-token", "")
	err = cEmpty.PhoneNumbers.Deregister(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}

func TestPhoneNumbersSetTwoStepPIN(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pin"] != "654321" {
			t.Errorf("unexpected set pin body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "123456", WithBaseURL(ts.URL))
	err := c.PhoneNumbers.SetTwoStepPIN(context.Background(), "phone-1", "654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty PIN error
	err = c.PhoneNumbers.SetTwoStepPIN(context.Background(), "phone-1", "")
	if err == nil {
		t.Fatal("expected error for empty PIN")
	}

	// Empty phone ID error
	cEmpty := New("test-token", "")
	err = cEmpty.PhoneNumbers.SetTwoStepPIN(context.Background(), "", "654321")
	if err == nil {
		t.Fatal("expected error for empty phone number ID")
	}
}
