package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrMissingSignature = errors.New("webhook: missing X-Hub-Signature-256 header")
	ErrInvalidSignature = errors.New("webhook: signature mismatch (invalid X-Hub-Signature-256)")
)

func VerifyChallenge(w http.ResponseWriter, r *http.Request, verifyToken string) bool {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == verifyToken {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return true
	}

	w.WriteHeader(http.StatusForbidden)
	return false
}

func VerifySignature(body []byte, signatureHeader string, appSecret string) error {
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return ErrInvalidSignature
	}
	expectedSigHex := parts[1]

	expectedSig, err := hex.DecodeString(expectedSigHex)
	if err != nil {
		return fmt.Errorf("webhook: failed to decode signature hex: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	actualSig := mac.Sum(nil)

	if !hmac.Equal(expectedSig, actualSig) {
		return ErrInvalidSignature
	}

	return nil
}

func ParsePayload(body []byte) (*Payload, error) {
	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("webhook: failed to parse payload JSON: %w", err)
	}
	return &payload, nil
}
