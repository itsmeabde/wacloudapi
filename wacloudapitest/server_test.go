package wacloudapitest_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/itsmeabde/wacloudapi/wacloudapitest"
)

func TestNewServer_Defaults(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	if srv.URL() == "" || !strings.HasPrefix(srv.URL(), "http://") {
		t.Fatalf("expected valid URL starting with http://, got %q", srv.URL())
	}

	if srv.PhoneNumberID() != wacloudapitest.DefaultPhoneNumberID {
		t.Errorf("expected default phone number ID %q, got %q", wacloudapitest.DefaultPhoneNumberID, srv.PhoneNumberID())
	}

	if srv.WABAID() != wacloudapitest.DefaultWABAID {
		t.Errorf("expected default WABA ID %q, got %q", wacloudapitest.DefaultWABAID, srv.WABAID())
	}

	if srv.APIVersion() != wacloudapitest.DefaultAPIVersion {
		t.Errorf("expected default API version %q, got %q", wacloudapitest.DefaultAPIVersion, srv.APIVersion())
	}

	if srv.AppSecret() != wacloudapitest.DefaultAppSecret {
		t.Errorf("expected default App Secret %q, got %q", wacloudapitest.DefaultAppSecret, srv.AppSecret())
	}

	if srv.AccessToken() != wacloudapitest.DefaultAccessToken {
		t.Errorf("expected default Access Token %q, got %q", wacloudapitest.DefaultAccessToken, srv.AccessToken())
	}

	client := srv.Client()
	if client == nil {
		t.Fatal("expected non-nil client from srv.Client()")
	}

	if srv.HTTPClient() == nil {
		t.Fatal("expected non-nil HTTP client from srv.HTTPClient()")
	}
}

func TestNewServer_CustomOptions(t *testing.T) {
	customPhoneID := "custom_phone_999"
	customWABAID := "custom_waba_888"
	customAPIVersion := "v22.0"
	customAppSecret := "custom_secret_777"
	customAccessToken := "custom_token_666"

	srv := wacloudapitest.NewServer(
		wacloudapitest.WithPhoneNumberID(customPhoneID),
		wacloudapitest.WithWABAID(customWABAID),
		wacloudapitest.WithAPIVersion(customAPIVersion),
		wacloudapitest.WithAppSecret(customAppSecret),
		wacloudapitest.WithAccessToken(customAccessToken),
	)
	defer srv.Close()

	if srv.PhoneNumberID() != customPhoneID {
		t.Errorf("expected phone number ID %q, got %q", customPhoneID, srv.PhoneNumberID())
	}

	if srv.WABAID() != customWABAID {
		t.Errorf("expected WABA ID %q, got %q", customWABAID, srv.WABAID())
	}

	if srv.APIVersion() != customAPIVersion {
		t.Errorf("expected API version %q, got %q", customAPIVersion, srv.APIVersion())
	}

	if srv.AppSecret() != customAppSecret {
		t.Errorf("expected App Secret %q, got %q", customAppSecret, srv.AppSecret())
	}

	if srv.AccessToken() != customAccessToken {
		t.Errorf("expected Access Token %q, got %q", customAccessToken, srv.AccessToken())
	}

	client := srv.Client()
	if client == nil {
		t.Fatal("expected non-nil client from srv.Client()")
	}
}

func TestNewServer_EmptyAndNilOptions(t *testing.T) {
	srv := wacloudapitest.NewServer(
		nil,
		wacloudapitest.WithPhoneNumberID(""),
		wacloudapitest.WithWABAID(""),
		wacloudapitest.WithAPIVersion(""),
		wacloudapitest.WithAppSecret(""),
		wacloudapitest.WithAccessToken(""),
	)
	defer srv.Close()

	if srv.PhoneNumberID() != wacloudapitest.DefaultPhoneNumberID {
		t.Errorf("expected default phone number ID when empty passed, got %q", srv.PhoneNumberID())
	}

	if srv.WABAID() != wacloudapitest.DefaultWABAID {
		t.Errorf("expected default WABA ID when empty passed, got %q", srv.WABAID())
	}

	if srv.APIVersion() != wacloudapitest.DefaultAPIVersion {
		t.Errorf("expected default API version when empty passed, got %q", srv.APIVersion())
	}

	if srv.AppSecret() != wacloudapitest.DefaultAppSecret {
		t.Errorf("expected default App Secret when empty passed, got %q", srv.AppSecret())
	}

	if srv.AccessToken() != wacloudapitest.DefaultAccessToken {
		t.Errorf("expected default Access Token when empty passed, got %q", srv.AccessToken())
	}
}

func TestServer_Reset(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	// Calling Reset should succeed without panic
	srv.Reset()
}

func TestServer_HTTPConnection(t *testing.T) {
	srv := wacloudapitest.NewServer()
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL()+"/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("failed to make request to mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("expected response body %q, got %q", `{"status":"ok"}`, string(body))
	}
}

func TestServer_Concurrency(t *testing.T) {
	srv := wacloudapitest.NewServer(
		wacloudapitest.WithPhoneNumberID("12345"),
		wacloudapitest.WithWABAID("67890"),
	)
	defer srv.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = srv.URL()
			_ = srv.PhoneNumberID()
			_ = srv.WABAID()
			_ = srv.APIVersion()
			_ = srv.AppSecret()
			_ = srv.AccessToken()
			_ = srv.Client()
			_ = srv.HTTPClient()
			if id%10 == 0 {
				srv.Reset()
			}
		}(i)
	}
	wg.Wait()
}
