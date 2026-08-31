package wacloudapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type QRCodesService struct {
	client *Client
}

func newQRCodesService(client *Client) *QRCodesService {
	return &QRCodesService{client: client}
}

func (s *QRCodesService) Create(ctx context.Context, req *CreateQRCodeRequest) (*QRCodeDetails, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: create qr code request cannot be nil")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls", s.client.config.PhoneNumberID)
	var resp QRCodeDetails
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) List(ctx context.Context) (*ListQRCodesResponse, error) {
	endpoint := fmt.Sprintf("%s/message_qrdls", s.client.config.PhoneNumberID)
	var resp ListQRCodesResponse
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) Get(ctx context.Context, qrCodeID string) (*QRCodeDetails, error) {
	if qrCodeID == "" {
		return nil, fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	var wrapper struct {
		Data []QRCodeDetails `json:"data"`
	}
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Data) == 0 {
		return nil, fmt.Errorf("wacloudapi: qr code not found")
	}
	return &wrapper.Data[0], nil
}

func (s *QRCodesService) Update(ctx context.Context, qrCodeID, prefilledMessage string) (*QRCodeDetails, error) {
	if qrCodeID == "" {
		return nil, fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	body := map[string]string{
		"prefilled_message": prefilledMessage,
	}
	var resp QRCodeDetails
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *QRCodesService) Delete(ctx context.Context, qrCodeID string) error {
	if qrCodeID == "" {
		return fmt.Errorf("wacloudapi: qrCodeID cannot be empty")
	}
	endpoint := fmt.Sprintf("%s/message_qrdls/%s", s.client.config.PhoneNumberID, qrCodeID)
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (s *QRCodesService) DownloadImage(ctx context.Context, qrCodeID string) ([]byte, error) {
	details, err := s.Get(ctx, qrCodeID)
	if err != nil {
		return nil, err
	}
	if details.QRImageURL == "" {
		return nil, fmt.Errorf("wacloudapi: qr_image_url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, details.QRImageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to create download request: %w", err)
	}

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to download QR image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wacloudapi: download QR image failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
