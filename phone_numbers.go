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
	if phoneNumberID == "" {
		return fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
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
	if phoneNumberID == "" {
		return fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
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
	if phoneNumberID == "" {
		return fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
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
	if phoneNumberID == "" {
		return fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
	}
	endpoint := fmt.Sprintf("%s/deregister", phoneNumberID)
	return s.client.sendJSON(ctx, http.MethodPost, endpoint, nil, nil)
}

func (s *PhoneNumbersService) SetTwoStepPIN(ctx context.Context, phoneNumberID, pin string) error {
	if phoneNumberID == "" {
		phoneNumberID = s.client.config.PhoneNumberID
	}
	if phoneNumberID == "" {
		return fmt.Errorf("wacloudapi: phoneNumberID cannot be empty")
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
