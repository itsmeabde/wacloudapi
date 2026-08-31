package wacloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediaServiceUpload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart content type, got %s", r.Header.Get("Content-Type"))
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		if r.FormValue("messaging_product") != "whatsapp" {
			t.Errorf("expected messaging_product whatsapp, got %s", r.FormValue("messaging_product"))
		}
		if r.FormValue("type") != "image/png" {
			t.Errorf("expected type image/png, got %s", r.FormValue("type"))
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get form file: %v", err)
		}
		defer file.Close()

		if handler.Filename != "sample.png" {
			t.Errorf("expected filename sample.png, got %s", handler.Filename)
		}

		_ = json.NewEncoder(w).Encode(UploadMediaResponse{ID: "media-upload-123"})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	dummyData := bytes.NewReader([]byte("fake image data"))
	res, err := c.Media.Upload(context.Background(), "sample.png", dummyData, "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "media-upload-123" {
		t.Errorf("expected id media-upload-123, got %s", res.ID)
	}
}

func TestMediaServiceUploadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(10 << 20)
		file, handler, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("failed to get form file: %v", err)
		}
		defer file.Close()

		content, _ := io.ReadAll(file)
		if string(content) != "temp file content" {
			t.Errorf("unexpected content: %s", string(content))
		}
		if handler.Filename != "test-file.txt" {
			t.Errorf("expected filename test-file.txt, got %s", handler.Filename)
		}

		_ = json.NewEncoder(w).Encode(UploadMediaResponse{ID: "media-file-456"})
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-file.txt")
	if err := os.WriteFile(tmpFile, []byte("temp file content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	res, err := c.Media.UploadFile(context.Background(), tmpFile, "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "media-file-456" {
		t.Errorf("expected id media-file-456, got %s", res.ID)
	}

	// Test non-existent file
	_, err = c.Media.UploadFile(context.Background(), filepath.Join(tmpDir, "non-existent.txt"), "text/plain")
	if err == nil {
		t.Errorf("expected error for non-existent file")
	}
}

func TestMediaServiceUpload_Errors(t *testing.T) {
	c := New("test-token", "12345")
	_, err := c.Media.Upload(context.Background(), "test.png", nil, "image/png")
	if err == nil {
		t.Errorf("expected error for nil reader")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(GraphErrorWrapper{
			Error: &APIError{
				Message: "Invalid file format",
				Type:    "OAuthException",
				Code:    100,
			},
		})
	}))
	defer ts.Close()

	c = New("test-token", "12345", WithBaseURL(ts.URL), WithRetry(0))
	_, err = c.Media.Upload(context.Background(), "test.png", strings.NewReader("bad data"), "image/png")
	if err == nil {
		t.Fatalf("expected error from API, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.HTTPStatusCode != http.StatusBadRequest || apiErr.Message != "Invalid file format" {
		t.Errorf("unexpected api error details: %+v", apiErr)
	}
}

func TestMediaServiceGetAndDownload(t *testing.T) {
	var downloadServer *httptest.Server
	downloadServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("binary file content"))
	}))
	defer downloadServer.Close()

	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/media-id-777" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(MediaMetadata{
			ID:               "media-id-777",
			URL:              downloadServer.URL,
			MimeType:         "image/jpeg",
			FileSize:         19,
			MessagingProduct: "whatsapp",
		})
	}))
	defer metaServer.Close()

	c := New("test-token", "12345", WithBaseURL(metaServer.URL))
	meta, err := c.Media.Get(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error getting meta: %v", err)
	}
	if meta.FileSize != 19 || meta.MimeType != "image/jpeg" {
		t.Errorf("unexpected meta: %+v", meta)
	}

	// Test streaming Download
	reader, downloadedMeta1, err := c.Media.Download(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error downloading stream: %v", err)
	}
	defer reader.Close()
	streamBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading stream: %v", err)
	}
	if string(streamBytes) != "binary file content" {
		t.Errorf("unexpected stream content: %s", string(streamBytes))
	}
	if downloadedMeta1.ID != "media-id-777" {
		t.Errorf("unexpected downloadedMeta1: %+v", downloadedMeta1)
	}

	// Test DownloadBytes
	data, downloadedMeta, err := c.Media.DownloadBytes(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error downloading bytes: %v", err)
	}
	if string(data) != "binary file content" {
		t.Errorf("unexpected downloaded data: %s", string(data))
	}
	if downloadedMeta.ID != "media-id-777" {
		t.Errorf("unexpected metadata from download: %+v", downloadedMeta)
	}
}

func TestMediaServiceDownload_Errors(t *testing.T) {
	c := New("test-token", "12345")
	_, _, err := c.Media.Download(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty media ID")
	}

	_, _, err = c.Media.DownloadBytes(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty media ID on DownloadBytes")
	}

	// Download server failure
	var downloadServer *httptest.Server
	downloadServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(GraphErrorWrapper{
			Error: &APIError{
				Message: "Media file expired or not found",
				Code:    404,
			},
		})
	}))
	defer downloadServer.Close()

	metaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(MediaMetadata{
			ID:  "media-expired",
			URL: downloadServer.URL,
		})
	}))
	defer metaServer.Close()

	c = New("test-token", "12345", WithBaseURL(metaServer.URL))
	_, _, err = c.Media.Download(context.Background(), "media-expired")
	if err == nil {
		t.Fatalf("expected error from download server, got nil")
	}
}

func TestMediaServiceGet_Errors(t *testing.T) {
	c := New("test-token", "12345")
	_, err := c.Media.Get(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty mediaID")
	}
}

func TestMediaServiceDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/v21.0/media-id-777" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	c := New("test-token", "12345", WithBaseURL(ts.URL))
	err := c.Media.Delete(context.Background(), "media-id-777")
	if err != nil {
		t.Fatalf("unexpected error deleting media: %v", err)
	}

	err = c.Media.Delete(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty mediaID")
	}
}
