package wacloudapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type MediaService struct {
	client *Client
}

func newMediaService(client *Client) *MediaService {
	return &MediaService{client: client}
}

func (s *MediaService) Upload(ctx context.Context, filename string, r io.Reader, mimeType string) (*UploadMediaResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("wacloudapi: media reader cannot be nil")
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		var err error
		defer func() {
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		}()
		if err = writer.WriteField("messaging_product", "whatsapp"); err != nil {
			return
		}
		if err = writer.WriteField("type", mimeType); err != nil {
			return
		}
		part, err2 := writer.CreateFormFile("file", filename)
		if err2 != nil {
			err = err2
			return
		}
		if _, err = io.Copy(part, r); err != nil {
			return
		}
		err = writer.Close()
	}()

	endpoint := fmt.Sprintf("%s/media", s.client.config.PhoneNumberID)
	url := s.client.buildURL(endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to create upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.client.config.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: media upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to read upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, s.client.parseAPIError(resp.StatusCode, respBytes)
	}

	var uploadResp UploadMediaResponse
	if err := json.Unmarshal(respBytes, &uploadResp); err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to unmarshal upload response: %w", err)
	}

	return &uploadResp, nil
}

func (s *MediaService) UploadFile(ctx context.Context, filePath, mimeType string) (*UploadMediaResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("wacloudapi: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	filename := filepath.Base(filePath)
	return s.Upload(ctx, filename, file, mimeType)
}

func (s *MediaService) Get(ctx context.Context, mediaID string) (*MediaMetadata, error) {
	if mediaID == "" {
		return nil, fmt.Errorf("wacloudapi: mediaID cannot be empty")
	}

	endpoint := mediaID
	var meta MediaMetadata
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *MediaService) Download(ctx context.Context, mediaID string) (io.ReadCloser, *MediaMetadata, error) {
	meta, err := s.Get(ctx, mediaID)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meta.URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: failed to create download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.client.config.AccessToken)

	resp, err := s.client.config.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: media download failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, nil, s.client.parseAPIError(resp.StatusCode, respBytes)
	}

	return resp.Body, meta, nil
}

func (s *MediaService) DownloadBytes(ctx context.Context, mediaID string) ([]byte, *MediaMetadata, error) {
	stream, meta, err := s.Download(ctx, mediaID)
	if err != nil {
		return nil, nil, err
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, nil, fmt.Errorf("wacloudapi: failed to read downloaded media bytes: %w", err)
	}
	return data, meta, nil
}

func (s *MediaService) Delete(ctx context.Context, mediaID string) error {
	if mediaID == "" {
		return fmt.Errorf("wacloudapi: mediaID cannot be empty")
	}
	endpoint := mediaID
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}
