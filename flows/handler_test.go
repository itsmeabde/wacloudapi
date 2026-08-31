package flows

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

// Helper function to create an encrypted request body and session keys for verification.
func createEncryptedRequest(t *testing.T, pubKey *rsa.PublicKey, req *Request) ([]byte, []byte, []byte) {
	t.Helper()

	aesKey := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		t.Fatalf("failed to generate AES key: %v", err)
	}

	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		t.Fatalf("failed to generate IV: %v", err)
	}

	encAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, aesKey, nil)
	if err != nil {
		t.Fatalf("failed to encrypt AES key: %v", err)
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("failed to create AES cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create GCM: %v", err)
	}
	encFlowData := gcm.Seal(nil, iv, reqBytes, nil)

	payload := EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString(encAESKey),
		EncryptedFlowData: base64.StdEncoding.EncodeToString(encFlowData),
		InitialVector:     base64.StdEncoding.EncodeToString(iv),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	return bodyBytes, aesKey, iv
}

// Helper function to decrypt response using AES key and flipped IV.
func decryptResponse(t *testing.T, aesKey, iv []byte, b64Response string) *Response {
	t.Helper()

	ciphertext, err := base64.StdEncoding.DecodeString(b64Response)
	if err != nil {
		t.Fatalf("failed to decode base64 response: %v", err)
	}

	flippedIV := make([]byte, len(iv))
	for i, b := range iv {
		flippedIV[i] = b ^ 0xFF
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("failed to create AES cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create GCM: %v", err)
	}

	plaintext, err := gcm.Open(nil, flippedIV, ciphertext, nil)
	if err != nil {
		t.Fatalf("failed to decrypt response: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(plaintext, &resp); err != nil {
		t.Fatalf("failed to unmarshal decrypted response: %v", err)
	}

	return &resp
}

func TestNewHandler_Constructors(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	})

	// 1. NewHandler
	h1 := NewHandler(key)
	if h1 == nil || h1.privateKey != key {
		t.Errorf("NewHandler failed")
	}

	// 2. NewHandlerFromPEM
	h2, err := NewHandlerFromPEM(pemBytes)
	if err != nil {
		t.Fatalf("NewHandlerFromPEM failed: %v", err)
	}
	if h2 == nil || !h2.privateKey.Equal(key) {
		t.Errorf("NewHandlerFromPEM key mismatch")
	}

	// NewHandlerFromPEM invalid
	if _, err := NewHandlerFromPEM([]byte("invalid pem")); err == nil {
		t.Errorf("expected error for invalid PEM, got nil")
	}

	// 3. NewHandlerFromKeyFile
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "flow_key.pem")
	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write temp key file: %v", err)
	}

	h3, err := NewHandlerFromKeyFile(keyPath)
	if err != nil {
		t.Fatalf("NewHandlerFromKeyFile failed: %v", err)
	}
	if h3 == nil || !h3.privateKey.Equal(key) {
		t.Errorf("NewHandlerFromKeyFile key mismatch")
	}

	// NewHandlerFromKeyFile non-existent
	if _, err := NewHandlerFromKeyFile(filepath.Join(tmpDir, "missing.pem")); err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	req := httptest.NewRequest(http.MethodGet, "/flows", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected HTTP 405, got %d", rr.Code)
	}
}

func TestHandler_InvalidBodyAndDecryptionError(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	var errReported error
	var errMu sync.Mutex
	h.OnError(func(ctx context.Context, err error) {
		errMu.Lock()
		errReported = err
		errMu.Unlock()
	})

	// 1. Invalid JSON body
	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader([]byte("not-json")))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400 for invalid json, got %d", rr.Code)
	}

	errMu.Lock()
	if errReported == nil {
		t.Errorf("expected OnError to be called for invalid json")
	}
	errReported = nil
	errMu.Unlock()

	// 2. Corrupted ciphertext payload
	badPayload := EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString([]byte("invalid")),
		EncryptedFlowData: base64.StdEncoding.EncodeToString([]byte("invalid")),
		InitialVector:     base64.StdEncoding.EncodeToString([]byte("123456789012")),
	}
	badPayloadBytes, _ := json.Marshal(badPayload)

	req = httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(badPayloadBytes))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP 400 for bad decryption, got %d", rr.Code)
	}

	errMu.Lock()
	if errReported == nil {
		t.Errorf("expected OnError to be called for decryption error")
	}
	errMu.Unlock()
}

func TestHandler_DefaultPing(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	pingReq := &Request{
		Version: "3.0",
		Action:  "ping",
	}
	body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, pingReq)

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decryptResponse(t, aesKey, iv, rr.Body.String())
	if resp.Version != "3.0" {
		t.Errorf("expected version 3.0, got %s", resp.Version)
	}
	if resp.Data["status"] != "active" {
		t.Errorf("expected status 'active', got %+v", resp.Data)
	}
}

func TestHandler_CustomOnPing(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	h.OnPing(func(ctx context.Context, req *Request) (*Response, error) {
		return &Response{
			Version: req.Version,
			Data: map[string]interface{}{
				"status": "custom_healthy",
				"uptime": float64(3600),
			},
		}, nil
	})

	pingReq := &Request{
		Version: "3.0",
		Action:  "ping",
	}
	body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, pingReq)

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rr.Code)
	}

	resp := decryptResponse(t, aesKey, iv, rr.Body.String())
	if resp.Data["status"] != "custom_healthy" || resp.Data["uptime"] != float64(3600) {
		t.Errorf("unexpected custom ping response: %+v", resp.Data)
	}
}

func TestHandler_HandleScreen_InitAndDataExchange(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	h.HandleScreen("APPOINTMENT", func(ctx context.Context, req *Request) (*Response, error) {
		if req.Action == "INIT" {
			return &Response{
				Screen: "APPOINTMENT",
				Data: map[string]interface{}{
					"slots": []interface{}{"09:00", "10:00", "11:00"},
				},
			}, nil
		}

		if req.Action == "data_exchange" {
			selectedSlot := req.Data["selected_slot"]
			return &Response{
				Screen: "CONFIRMATION",
				Data: map[string]interface{}{
					"confirmed_slot": selectedSlot,
					"booking_id":     "BOOK-987",
				},
			}, nil
		}

		return nil, errors.New("unsupported action for APPOINTMENT screen")
	})

	// 1. Test INIT request
	initReq := &Request{
		Version:   "3.0",
		Action:    "INIT",
		Screen:    "APPOINTMENT",
		FlowToken: "token_xyz",
	}
	body1, aesKey1, iv1 := createEncryptedRequest(t, &key.PublicKey, initReq)

	httpReq1 := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body1))
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, httpReq1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 on INIT, got %d", rr1.Code)
	}
	resp1 := decryptResponse(t, aesKey1, iv1, rr1.Body.String())
	if resp1.Screen != "APPOINTMENT" {
		t.Errorf("expected screen APPOINTMENT, got %s", resp1.Screen)
	}
	slots, ok := resp1.Data["slots"].([]interface{})
	if !ok || len(slots) != 3 {
		t.Errorf("expected 3 slots, got %+v", resp1.Data["slots"])
	}

	// 2. Test data_exchange request
	exchangeReq := &Request{
		Version:   "3.0",
		Action:    "data_exchange",
		Screen:    "APPOINTMENT",
		FlowToken: "token_xyz",
		Data: map[string]interface{}{
			"selected_slot": "10:00",
		},
	}
	body2, aesKey2, iv2 := createEncryptedRequest(t, &key.PublicKey, exchangeReq)

	httpReq2 := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body2))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, httpReq2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 on data_exchange, got %d", rr2.Code)
	}
	resp2 := decryptResponse(t, aesKey2, iv2, rr2.Body.String())
	if resp2.Screen != "CONFIRMATION" {
		t.Errorf("expected screen CONFIRMATION, got %s", resp2.Screen)
	}
	if resp2.Data["confirmed_slot"] != "10:00" || resp2.Data["booking_id"] != "BOOK-987" {
		t.Errorf("unexpected confirmation response data: %+v", resp2.Data)
	}
}

func TestHandler_OnActionFallback(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	var actionCalled bool
	h.OnAction(func(ctx context.Context, req *Request) (*Response, error) {
		actionCalled = true
		return &Response{
			Screen: "FALLBACK_SCREEN",
			Data: map[string]interface{}{
				"handled_action": req.Action,
			},
		}, nil
	})

	flowReq := &Request{
		Version: "3.0",
		Action:  "custom_navigate",
		Screen:  "UNKNOWN_SCREEN",
	}
	body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, flowReq)

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rr.Code)
	}
	if !actionCalled {
		t.Errorf("expected OnAction fallback to be called")
	}

	resp := decryptResponse(t, aesKey, iv, rr.Body.String())
	if resp.Screen != "FALLBACK_SCREEN" || resp.Data["handled_action"] != "custom_navigate" {
		t.Errorf("unexpected fallback response: %+v", resp)
	}
}

func TestHandler_ScreenHandlerReturnsError(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	var loggedError error
	h.OnError(func(ctx context.Context, err error) {
		loggedError = err
	})

	h.HandleScreen("ERROR_SCREEN", func(ctx context.Context, req *Request) (*Response, error) {
		return nil, errors.New("database connection failed")
	})

	flowReq := &Request{
		Version: "3.0",
		Action:  "data_exchange",
		Screen:  "ERROR_SCREEN",
	}
	body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, flowReq)

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 even on flow error response, got %d", rr.Code)
	}

	if loggedError == nil || loggedError.Error() != "database connection failed" {
		t.Errorf("expected OnError to catch handler error, got %v", loggedError)
	}

	resp := decryptResponse(t, aesKey, iv, rr.Body.String())
	if resp.Error == nil || resp.Error.Message != "database connection failed" {
		t.Errorf("expected flow error in decrypted response, got %+v", resp)
	}
}

func TestHandler_UnhandledScreenAndAction(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	var loggedErr error
	h.OnError(func(ctx context.Context, err error) {
		loggedErr = err
	})

	flowReq := &Request{
		Version: "3.0",
		Action:  "unhandled_action",
		Screen:  "UNHANDLED_SCREEN",
	}
	body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, flowReq)

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rr.Code)
	}

	if loggedErr == nil {
		t.Errorf("expected error logged for unhandled screen/action")
	}

	resp := decryptResponse(t, aesKey, iv, rr.Body.String())
	if resp.Error == nil {
		t.Errorf("expected flow error in response for unhandled request")
	}
}

func TestHandler_Concurrency(t *testing.T) {
	key := generateTestRSAKey(t)
	h := NewHandler(key)

	h.HandleScreen("CONCURRENT_SCREEN", func(ctx context.Context, req *Request) (*Response, error) {
		id := req.Data["id"]
		return &Response{
			Screen: "CONCURRENT_SCREEN",
			Data: map[string]interface{}{
				"echo_id": id,
			},
		}, nil
	})

	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			flowReq := &Request{
				Version: "3.0",
				Action:  "data_exchange",
				Screen:  "CONCURRENT_SCREEN",
				Data: map[string]interface{}{
					"id": float64(idx),
				},
			}
			body, aesKey, iv := createEncryptedRequest(t, &key.PublicKey, flowReq)

			req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("worker %d: expected HTTP 200, got %d", idx, rr.Code)
				return
			}

			resp := decryptResponse(t, aesKey, iv, rr.Body.String())
			if !reflect.DeepEqual(resp.Data["echo_id"], float64(idx)) {
				t.Errorf("worker %d: expected echo_id %d, got %v", idx, idx, resp.Data["echo_id"])
			}
			atomic.AddInt64(&counter, 1)
		}(i)
	}

	wg.Wait()

	if counter != 30 {
		t.Errorf("expected 30 requests processed, got %d", counter)
	}
}

func TestHandler_NilPrivateKey(t *testing.T) {
	h := NewHandler(nil)

	var loggedErr error
	h.OnError(func(ctx context.Context, err error) {
		loggedErr = err
	})

	req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewReader([]byte("{}")))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500 for nil private key, got %d", rr.Code)
	}
	if loggedErr == nil {
		t.Errorf("expected OnError to be called for nil private key")
	}
}

func TestNewHandlerFromPEM_Encrypted(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes := x509.MarshalPKCS1PrivateKey(key)
	passphrase := "secret-flow-pass"

	//nolint:staticcheck
	encBlock, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", derBytes, []byte(passphrase), x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("failed to encrypt PEM block: %v", err)
	}
	pemBytes := pem.EncodeToMemory(encBlock)

	h, err := NewHandlerFromPEM(pemBytes, passphrase)
	if err != nil {
		t.Fatalf("NewHandlerFromPEM failed with passphrase: %v", err)
	}
	if h == nil || !h.privateKey.Equal(key) {
		t.Errorf("NewHandlerFromPEM private key mismatch")
	}
}

