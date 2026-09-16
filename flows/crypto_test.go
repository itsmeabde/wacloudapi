package flows

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// Helper to generate a test RSA Private Key
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func TestParsePrivateKey_PKCS1(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	parsedKey, err := ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed for PKCS#1: %v", err)
	}

	if !key.Equal(parsedKey) {
		t.Errorf("parsed key does not match original key")
	}
}

func TestParsePrivateKey_PKCS8(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal PKCS#8 private key: %v", err)
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	parsedKey, err := ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed for PKCS#8: %v", err)
	}

	if !key.Equal(parsedKey) {
		t.Errorf("parsed key does not match original key")
	}
}

func TestParsePrivateKey_EncryptedPKCS1(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes := x509.MarshalPKCS1PrivateKey(key)
	passphrase := "secret123"

	// Encrypt PEM block
	//nolint:staticcheck // testing legacy encrypted PEM block support
	encBlock, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", derBytes, []byte(passphrase), x509.PEMCipherAES256)
	if err != nil {
		t.Fatalf("failed to encrypt PEM block: %v", err)
	}
	pemBytes := pem.EncodeToMemory(encBlock)

	// Test with correct passphrase
	parsedKey, err := ParsePrivateKey(pemBytes, passphrase)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed with correct passphrase: %v", err)
	}
	if !key.Equal(parsedKey) {
		t.Errorf("parsed key does not match original key")
	}

	// Test with wrong passphrase
	_, err = ParsePrivateKey(pemBytes, "wrongpassword")
	if err == nil {
		t.Errorf("expected error with wrong passphrase, got nil")
	}

	// Test with missing passphrase
	_, err = ParsePrivateKey(pemBytes)
	if err == nil {
		t.Errorf("expected error with missing passphrase, got nil")
	}
}

func TestParsePrivateKey_InvalidInputs(t *testing.T) {
	// Empty bytes
	if _, err := ParsePrivateKey([]byte("")); err == nil {
		t.Errorf("expected error for empty bytes, got nil")
	}

	// Invalid PEM format
	if _, err := ParsePrivateKey([]byte("not a pem block")); err == nil {
		t.Errorf("expected error for non-pem bytes, got nil")
	}

	// Non-key PEM block
	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte("fake cert data"),
	}
	if _, err := ParsePrivateKey(pem.EncodeToMemory(certBlock)); err == nil {
		t.Errorf("expected error for certificate block, got nil")
	}
}

func TestLoadPrivateKeyFile(t *testing.T) {
	key := generateTestRSAKey(t)
	derBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	})

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_private.pem")
	if err := os.WriteFile(keyFile, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write temp key file: %v", err)
	}

	loadedKey, err := LoadPrivateKeyFile(keyFile)
	if err != nil {
		t.Fatalf("LoadPrivateKeyFile failed: %v", err)
	}
	if !key.Equal(loadedKey) {
		t.Errorf("loaded key does not match original")
	}

	// Test non-existent file
	if _, err := LoadPrivateKeyFile(filepath.Join(tmpDir, "non_existent.pem")); err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

func TestDecryptRequest_And_EncryptResponse(t *testing.T) {
	rsaKey := generateTestRSAKey(t)

	// 1. Prepare simulated WhatsApp Client payload
	aesKey := make([]byte, 16) // AES-128
	if _, err := rand.Read(aesKey); err != nil {
		t.Fatalf("failed to generate AES key: %v", err)
	}

	iv := make([]byte, 12) // 12-byte GCM nonce
	if _, err := rand.Read(iv); err != nil {
		t.Fatalf("failed to generate IV: %v", err)
	}

	// Encrypt AES key using RSA-OAEP SHA-256
	encAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &rsaKey.PublicKey, aesKey, nil)
	if err != nil {
		t.Fatalf("failed to encrypt AES key with RSA-OAEP: %v", err)
	}

	reqBody := Request{
		Version:   "3.0",
		Action:    "INIT",
		Screen:    "FIRST_SCREEN",
		FlowToken: "token_abc_123",
		Data: map[string]interface{}{
			"user_id":   "user_999",
			"is_member": true,
		},
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	// Encrypt request with AES-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("failed to create AES cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create GCM: %v", err)
	}
	encFlowData := gcm.Seal(nil, iv, reqBytes, nil)

	payload := &EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString(encAESKey),
		EncryptedFlowData: base64.StdEncoding.EncodeToString(encFlowData),
		InitialVector:     base64.StdEncoding.EncodeToString(iv),
	}

	// 2. Decrypt using DecryptRequest
	decryptedReq, session, err := DecryptRequest(payload, rsaKey)
	if err != nil {
		t.Fatalf("DecryptRequest failed: %v", err)
	}

	if decryptedReq.Version != reqBody.Version {
		t.Errorf("expected version %s, got %s", reqBody.Version, decryptedReq.Version)
	}
	if decryptedReq.Action != reqBody.Action {
		t.Errorf("expected action %s, got %s", reqBody.Action, decryptedReq.Action)
	}
	if decryptedReq.Screen != reqBody.Screen {
		t.Errorf("expected screen %s, got %s", reqBody.Screen, decryptedReq.Screen)
	}
	if decryptedReq.FlowToken != reqBody.FlowToken {
		t.Errorf("expected flow_token %s, got %s", reqBody.FlowToken, decryptedReq.FlowToken)
	}
	if !reflect.DeepEqual(decryptedReq.Data["user_id"], "user_999") || !reflect.DeepEqual(decryptedReq.Data["is_member"], true) {
		t.Errorf("unexpected decrypted data: %+v", decryptedReq.Data)
	}
	if !reflect.DeepEqual(session.AESKey, aesKey) {
		t.Errorf("session AES key mismatch")
	}
	if !reflect.DeepEqual(session.InitialVector, iv) {
		t.Errorf("session IV mismatch")
	}

	// 3. Encrypt Response using EncryptResponse
	resp := &Response{
		Version: "3.0",
		Screen:  "SUCCESS_SCREEN",
		Data: map[string]interface{}{
			"status":      "ok",
			"appointment": "2026-09-01",
		},
	}

	encRespB64, err := EncryptResponse(resp, session)
	if err != nil {
		t.Fatalf("EncryptResponse failed: %v", err)
	}

	// 4. Verify WhatsApp Client side decryption (with flipped IV)
	respCiphertext, err := base64.StdEncoding.DecodeString(encRespB64)
	if err != nil {
		t.Fatalf("failed to decode response base64: %v", err)
	}

	flippedIV := make([]byte, len(iv))
	for i, b := range iv {
		flippedIV[i] = b ^ 0xFF
	}

	respPlaintext, err := gcm.Open(nil, flippedIV, respCiphertext, nil)
	if err != nil {
		t.Fatalf("failed to decrypt response with flipped IV: %v", err)
	}

	var parsedResp Response
	if err := json.Unmarshal(respPlaintext, &parsedResp); err != nil {
		t.Fatalf("failed to unmarshal response plaintext: %v", err)
	}

	if parsedResp.Screen != resp.Screen {
		t.Errorf("expected response screen %s, got %s", resp.Screen, parsedResp.Screen)
	}
	if parsedResp.Data["status"] != "ok" || parsedResp.Data["appointment"] != "2026-09-01" {
		t.Errorf("unexpected response data: %+v", parsedResp.Data)
	}
}

func TestDecryptRequest_Errors(t *testing.T) {
	rsaKey := generateTestRSAKey(t)

	// Nil payload
	if _, _, err := DecryptRequest(nil, rsaKey); err == nil {
		t.Errorf("expected error for nil payload, got nil")
	}

	// Nil key
	if _, _, err := DecryptRequest(&EncryptedPayload{}, nil); err == nil {
		t.Errorf("expected error for nil key, got nil")
	}

	// Invalid Base64 for AES key
	payload := &EncryptedPayload{
		EncryptedAESKey:   "invalid-base64!",
		EncryptedFlowData: base64.StdEncoding.EncodeToString([]byte("data")),
		InitialVector:     base64.StdEncoding.EncodeToString([]byte("123456789012")),
	}
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for invalid AES key base64, got nil")
	}

	// Invalid Base64 for Flow data
	payload = &EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString([]byte("key")),
		EncryptedFlowData: "invalid-base64!",
		InitialVector:     base64.StdEncoding.EncodeToString([]byte("123456789012")),
	}
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for invalid flow data base64, got nil")
	}

	// Invalid Base64 for IV
	payload = &EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString([]byte("key")),
		EncryptedFlowData: base64.StdEncoding.EncodeToString([]byte("data")),
		InitialVector:     "invalid-base64!",
	}
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for invalid IV base64, got nil")
	}

	// Invalid IV length (8 bytes instead of 12) - should return error safely without panic
	validEncAESKey, _ := rsa.EncryptOAEP(sha256.New(), rand.Reader, &rsaKey.PublicKey, make([]byte, 16), nil)
	payload = &EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString(validEncAESKey),
		EncryptedFlowData: base64.StdEncoding.EncodeToString([]byte("data")),
		InitialVector:     base64.StdEncoding.EncodeToString([]byte("12345678")), // 8 bytes
	}
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for 8-byte IV in DecryptRequest, got nil")
	}

	// Invalid IV length (16 bytes instead of 12)
	payload.InitialVector = base64.StdEncoding.EncodeToString([]byte("1234567890123456"))
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for 16-byte IV in DecryptRequest, got nil")
	}

	// RSA decryption failure (corrupted ciphertext)
	payload = &EncryptedPayload{
		EncryptedAESKey:   base64.StdEncoding.EncodeToString([]byte("not an rsa ciphertext")),
		EncryptedFlowData: base64.StdEncoding.EncodeToString([]byte("data")),
		InitialVector:     base64.StdEncoding.EncodeToString([]byte("123456789012")),
	}
	if _, _, err := DecryptRequest(payload, rsaKey); err == nil {
		t.Errorf("expected error for failed RSA decryption, got nil")
	}
}

func TestEncryptResponse_Errors(t *testing.T) {
	session := &CryptoSession{
		AESKey:        make([]byte, 16),
		InitialVector: make([]byte, 12),
	}

	// Nil response
	if _, err := EncryptResponse(nil, session); err == nil {
		t.Errorf("expected error for nil response, got nil")
	}

	// Nil session
	if _, err := EncryptResponse(&Response{Screen: "A"}, nil); err == nil {
		t.Errorf("expected error for nil session, got nil")
	}

	// Empty AES key
	if _, err := EncryptResponse(&Response{Screen: "A"}, &CryptoSession{AESKey: nil, InitialVector: make([]byte, 12)}); err == nil {
		t.Errorf("expected error for empty AES key, got nil")
	}

	// Empty IV
	if _, err := EncryptResponse(&Response{Screen: "A"}, &CryptoSession{AESKey: make([]byte, 16), InitialVector: nil}); err == nil {
		t.Errorf("expected error for empty IV, got nil")
	}

	// Invalid IV length (8 bytes)
	if _, err := EncryptResponse(&Response{Screen: "A"}, &CryptoSession{AESKey: make([]byte, 16), InitialVector: make([]byte, 8)}); err == nil {
		t.Errorf("expected error for 8-byte IV, got nil")
	}

	// Invalid IV length (16 bytes)
	if _, err := EncryptResponse(&Response{Screen: "A"}, &CryptoSession{AESKey: make([]byte, 16), InitialVector: make([]byte, 16)}); err == nil {
		t.Errorf("expected error for 16-byte IV, got nil")
	}
}

func TestConcurrency_DecryptAndEncrypt(t *testing.T) {
	rsaKey := generateTestRSAKey(t)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			aesKey := make([]byte, 16)
			_, _ = rand.Read(aesKey)
			iv := make([]byte, 12)
			_, _ = rand.Read(iv)

			encAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &rsaKey.PublicKey, aesKey, nil)
			if err != nil {
				t.Errorf("worker %d: failed to encrypt AES key: %v", idx, err)
				return
			}

			reqBody := Request{
				Version:   "3.0",
				Action:    "data_exchange",
				Screen:    "SCREEN_A",
				FlowToken: "token",
				Data: map[string]interface{}{
					"index": float64(idx),
				},
			}
			reqBytes, _ := json.Marshal(reqBody)

			block, _ := aes.NewCipher(aesKey)
			gcm, _ := cipher.NewGCM(block)
			encFlowData := gcm.Seal(nil, iv, reqBytes, nil)

			payload := &EncryptedPayload{
				EncryptedAESKey:   base64.StdEncoding.EncodeToString(encAESKey),
				EncryptedFlowData: base64.StdEncoding.EncodeToString(encFlowData),
				InitialVector:     base64.StdEncoding.EncodeToString(iv),
			}

			decrypted, session, err := DecryptRequest(payload, rsaKey)
			if err != nil {
				t.Errorf("worker %d: DecryptRequest failed: %v", idx, err)
				return
			}

			if decrypted.Action != "data_exchange" {
				t.Errorf("worker %d: unexpected action: %s", idx, decrypted.Action)
			}

			resp := &Response{
				Screen: "SCREEN_B",
				Data: map[string]interface{}{
					"result": float64(idx * 2),
				},
			}
			encResp, err := EncryptResponse(resp, session)
			if err != nil {
				t.Errorf("worker %d: EncryptResponse failed: %v", idx, err)
				return
			}
			if encResp == "" {
				t.Errorf("worker %d: empty encrypted response", idx)
			}
		}(i)
	}

	wg.Wait()
}
