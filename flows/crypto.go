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
	"errors"
	"fmt"
	"os"
)

// ParsePrivateKey parses an RSA private key from PEM-encoded bytes.
// It supports PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") formats,
// as well as password-protected (encrypted) PEM blocks if a passphrase is provided.
func ParsePrivateKey(pemBytes []byte, passphrase ...string) (*rsa.PrivateKey, error) {
	if len(pemBytes) == 0 {
		return nil, errors.New("pemBytes cannot be empty")
	}

	var block *pem.Block
	rest := pemBytes

	for {
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		derBytes := block.Bytes

		// Check if block is encrypted
		//nolint:staticcheck // support legacy encrypted PEM blocks
		if x509.IsEncryptedPEMBlock(block) || block.Headers["Proc-Type"] == "4,ENCRYPTED" {
			if len(passphrase) == 0 || passphrase[0] == "" {
				return nil, errors.New("passphrase required for encrypted private key")
			}
			//nolint:staticcheck // support legacy encrypted PEM blocks
			decrypted, err := x509.DecryptPEMBlock(block, []byte(passphrase[0]))
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt PEM block: %w", err)
			}
			derBytes = decrypted
		}

		// Try parsing as PKCS#1
		if privKey, err := x509.ParsePKCS1PrivateKey(derBytes); err == nil {
			return privKey, nil
		}

		// Try parsing as PKCS#8
		if key, err := x509.ParsePKCS8PrivateKey(derBytes); err == nil {
			if privKey, ok := key.(*rsa.PrivateKey); ok {
				return privKey, nil
			}
			return nil, errors.New("parsed PKCS#8 private key is not an RSA private key")
		}
	}

	// If no PEM block decoded or invalid block, try parsing raw bytes directly as PKCS#1/PKCS#8 DER
	if privKey, err := x509.ParsePKCS1PrivateKey(pemBytes); err == nil {
		return privKey, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(pemBytes); err == nil {
		if privKey, ok := key.(*rsa.PrivateKey); ok {
			return privKey, nil
		}
		return nil, errors.New("raw DER private key is not an RSA private key")
	}

	return nil, errors.New("no valid RSA private key found in provided PEM data")
}

// LoadPrivateKeyFile reads an RSA private key from the given file path.
// It accepts an optional passphrase for encrypted private keys.
func LoadPrivateKeyFile(path string, passphrase ...string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	return ParsePrivateKey(data, passphrase...)
}

// DecryptRequest decrypts an incoming WhatsApp Flows encrypted request payload.
// It decrypts the AES key using RSA-OAEP with SHA-256 and the provided RSA private key,
// then decrypts the flow data using AES-128-GCM with the initial vector.
// It returns the parsed Request, a CryptoSession for response encryption, or an error.
func DecryptRequest(payload *EncryptedPayload, key *rsa.PrivateKey) (*Request, *CryptoSession, error) {
	if payload == nil {
		return nil, nil, errors.New("payload cannot be nil")
	}
	if key == nil {
		return nil, nil, errors.New("private key cannot be nil")
	}

	encAESKey, err := base64.StdEncoding.DecodeString(payload.EncryptedAESKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode encrypted_aes_key base64: %w", err)
	}

	encFlowData, err := base64.StdEncoding.DecodeString(payload.EncryptedFlowData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode encrypted_flow_data base64: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(payload.InitialVector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode initial_vector base64: %w", err)
	}

	// Decrypt AES key with RSA-OAEP SHA-256
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encAESKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt AES key with RSA-OAEP: %w", err)
	}

	// Setup AES-GCM cipher
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	// Decrypt flow data
	plaintext, err := gcm.Open(nil, iv, encFlowData, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt flow data with AES-GCM: %w", err)
	}

	var req Request
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal decrypted flow request: %w", err)
	}

	session := &CryptoSession{
		AESKey:        aesKey,
		InitialVector: iv,
	}

	return &req, session, nil
}

// EncryptResponse encrypts a WhatsApp Flows Response using AES-128-GCM with the
// session's AES key and an inverted initial vector (each byte of IV XORed with 0xFF).
// It returns a Base64-encoded ciphertext string ready to be sent to WhatsApp.
func EncryptResponse(resp *Response, session *CryptoSession) (string, error) {
	if resp == nil {
		return "", errors.New("response cannot be nil")
	}
	if session == nil {
		return "", errors.New("crypto session cannot be nil")
	}
	if len(session.AESKey) == 0 {
		return "", errors.New("session AES key is empty")
	}
	if len(session.InitialVector) == 0 {
		return "", errors.New("session initial vector is empty")
	}

	plaintext, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response payload: %w", err)
	}

	// Invert the IV: flipped_iv[i] = iv[i] ^ 0xFF
	flippedIV := make([]byte, len(session.InitialVector))
	for i, b := range session.InitialVector {
		flippedIV[i] = b ^ 0xFF
	}

	block, err := aes.NewCipher(session.AESKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	ciphertext := gcm.Seal(nil, flippedIV, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
