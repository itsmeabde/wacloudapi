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

func TestBusinessProfileGetCustomFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/12345/whatsapp_business_profile" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("fields") != "about,email" {
			t.Errorf("unexpected fields query: %s", r.URL.Query().Get("fields"))
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []BusinessProfile{
				{
					About: "About company",
					Email: "contact@company.com",
				},
			},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	profile, err := c.BusinessProfile.Get(context.Background(), "about", "email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.About != "About company" || profile.Email != "contact@company.com" {
		t.Errorf("unexpected profile: %+v", profile)
	}
}

func TestBusinessProfileGetEmptyData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []BusinessProfile{},
		})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	profile, err := c.BusinessProfile.Get(context.Background())
	if err == nil {
		t.Fatalf("expected error for empty data, got profile: %+v", profile)
	}
}

func TestBusinessProfileUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/v21.0/12345/whatsapp_business_profile" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req UpdateBusinessProfileRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.MessagingProduct != "whatsapp" {
			t.Errorf("expected messaging_product whatsapp, got %s", req.MessagingProduct)
		}
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

func TestBusinessProfileUpdateNilRequest(t *testing.T) {
	c := New("test-token", "12345")
	err := c.BusinessProfile.Update(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request, got nil")
	}
}
