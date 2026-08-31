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
			var body CreateQRCodeRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.PrefilledMessage != "Hello from QR" || body.ImageFormat != QRCodeImageSVG {
				t.Errorf("unexpected create body: %+v", body)
			}
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

func TestQRCodesServiceCreate_NilReq(t *testing.T) {
	c := New("test-token", "phone-1")
	_, err := c.QRCodes.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when req is nil")
	}
}

func TestQRCodesServiceList(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/message_qrdls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(ListQRCodesResponse{
			Data: []QRCodeDetails{
				{
					Code:             "QR_1",
					PrefilledMessage: "Msg 1",
					DeepLinkURL:      "https://wa.me/message/QR_1",
					QRImageURL:       "https://example.com/qr1.png",
				},
				{
					Code:             "QR_2",
					PrefilledMessage: "Msg 2",
					DeepLinkURL:      "https://wa.me/message/QR_2",
					QRImageURL:       "https://example.com/qr2.png",
				},
			},
		})
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))
	res, err := c.QRCodes.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(res.Data) != 2 {
		t.Fatalf("expected 2 qr codes, got %d", len(res.Data))
	}
	if res.Data[0].Code != "QR_1" || res.Data[1].Code != "QR_2" {
		t.Errorf("unexpected qr codes: %+v", res.Data)
	}
}

func TestQRCodesServiceGet(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/message_qrdls/QR_999" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(struct {
			Data []QRCodeDetails `json:"data"`
		}{
			Data: []QRCodeDetails{
				{
					Code:             "QR_999",
					PrefilledMessage: "Special Offer",
					DeepLinkURL:      "https://wa.me/message/QR_999",
					QRImageURL:       "https://example.com/qr999.png",
				},
			},
		})
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))
	qr, err := c.QRCodes.Get(context.Background(), "QR_999")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if qr.Code != "QR_999" || qr.PrefilledMessage != "Special Offer" {
		t.Errorf("unexpected qr: %+v", qr)
	}
}

func TestQRCodesServiceGet_Errors(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Data []QRCodeDetails `json:"data"`
		}{
			Data: []QRCodeDetails{},
		})
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))

	// Empty ID
	_, err := c.QRCodes.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty qrCodeID")
	}

	// Empty data array / Not found
	_, err = c.QRCodes.Get(context.Background(), "NON_EXISTENT")
	if err == nil {
		t.Fatal("expected error when qr code data is empty")
	}
}

func TestQRCodesServiceUpdate(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/message_qrdls/QR_123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["prefilled_message"] != "Updated Message" {
			t.Errorf("unexpected prefilled_message: %s", body["prefilled_message"])
		}
		_ = json.NewEncoder(w).Encode(QRCodeDetails{
			Code:             "QR_123",
			PrefilledMessage: "Updated Message",
			DeepLinkURL:      "https://wa.me/message/QR_123",
			QRImageURL:       "https://example.com/qr123.png",
		})
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))
	qr, err := c.QRCodes.Update(context.Background(), "QR_123", "Updated Message")
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if qr.PrefilledMessage != "Updated Message" {
		t.Errorf("unexpected prefilled_message: %s", qr.PrefilledMessage)
	}

	// Empty ID error
	_, err = c.QRCodes.Update(context.Background(), "", "Updated Message")
	if err == nil {
		t.Fatal("expected error for empty qrCodeID")
	}
}

func TestQRCodesServiceDelete(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-1/message_qrdls/QR_123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))
	err := c.QRCodes.Delete(context.Background(), "QR_123")
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	// Empty ID error
	err = c.QRCodes.Delete(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty qrCodeID")
	}
}

func TestQRCodesServiceDownloadImage_Errors(t *testing.T) {
	var handler func(w http.ResponseWriter, r *http.Request)
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer apiServer.Close()

	c := New("test-token", "phone-1", WithBaseURL(apiServer.URL))

	// 1. Get fails (empty ID)
	_, err := c.QRCodes.DownloadImage(context.Background(), "")
	if err == nil {
		t.Fatal("expected error on empty qrCodeID")
	}

	// 2. QRImageURL is empty
	handler = func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Data []QRCodeDetails `json:"data"`
		}{
			Data: []QRCodeDetails{
				{
					Code:             "QR_NO_IMG",
					PrefilledMessage: "No image",
					QRImageURL:       "",
				},
			},
		})
	}
	_, err = c.QRCodes.DownloadImage(context.Background(), "QR_NO_IMG")
	if err == nil {
		t.Fatal("expected error when qr_image_url is empty")
	}

	// 3. Download returns HTTP error status (500)
	imgErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer imgErrorServer.Close()

	handler = func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Data []QRCodeDetails `json:"data"`
		}{
			Data: []QRCodeDetails{
				{
					Code:             "QR_ERR",
					PrefilledMessage: "Error download",
					QRImageURL:       imgErrorServer.URL,
				},
			},
		})
	}
	_, err = c.QRCodes.DownloadImage(context.Background(), "QR_ERR")
	if err == nil {
		t.Fatal("expected error when image download returns 500 status")
	}
}
