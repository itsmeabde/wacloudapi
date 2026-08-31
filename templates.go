package wacloudapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type TemplatesService struct {
	client *Client
}

func newTemplatesService(client *Client) *TemplatesService {
	return &TemplatesService{client: client}
}

func (s *TemplatesService) resolveWABAID(explicitID string) (string, error) {
	if explicitID != "" {
		return explicitID, nil
	}
	if s.client.config.WABAID != "" {
		return s.client.config.WABAID, nil
	}
	return "", fmt.Errorf("wacloudapi: WABA ID is required for template operations (use WithWABAID() option or specify in request)")
}

func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*CreateTemplateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: request cannot be nil")
	}

	wabaID, err := s.resolveWABAID(req.WABAID)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/message_templates", wabaID)
	var resp CreateTemplateResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) List(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	var explicitWABA string
	params := url.Values{}
	if req != nil {
		explicitWABA = req.WABAID
		if req.Category != "" {
			params.Set("category", string(req.Category))
		}
		if req.Status != "" {
			params.Set("status", string(req.Status))
		}
		if req.Name != "" {
			params.Set("name", req.Name)
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.Itoa(req.Limit))
		}
		if req.After != "" {
			params.Set("after", req.After)
		}
		if req.Before != "" {
			params.Set("before", req.Before)
		}
	}

	wabaID, err := s.resolveWABAID(explicitWABA)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/message_templates", wabaID)
	if q := params.Encode(); q != "" {
		endpoint += "?" + q
	}

	var resp ListTemplatesResponse
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) NextPage(ctx context.Context, current *ListTemplatesResponse, wabaID ...string) (*ListTemplatesResponse, error) {
	if current == nil || !current.HasNext() {
		return nil, fmt.Errorf("wacloudapi: no next page available")
	}
	var explicitWABA string
	if len(wabaID) > 0 {
		explicitWABA = wabaID[0]
	}
	return s.List(ctx, &ListTemplatesRequest{
		WABAID: explicitWABA,
		After:  current.NextCursor(),
	})
}

func (s *TemplatesService) PrevPage(ctx context.Context, current *ListTemplatesResponse, wabaID ...string) (*ListTemplatesResponse, error) {
	if current == nil || !current.HasPrevious() {
		return nil, fmt.Errorf("wacloudapi: no previous page available")
	}
	var explicitWABA string
	if len(wabaID) > 0 {
		explicitWABA = wabaID[0]
	}
	return s.List(ctx, &ListTemplatesRequest{
		WABAID: explicitWABA,
		Before: current.PreviousCursor(),
	})
}

func (s *TemplatesService) Get(ctx context.Context, templateID string) (*TemplateDetails, error) {
	if templateID == "" {
		return nil, fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	endpoint := templateID
	var details TemplateDetails
	if err := s.client.sendJSON(ctx, http.MethodGet, endpoint, nil, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

func (s *TemplatesService) Update(ctx context.Context, templateID string, req *UpdateTemplateRequest) (*UpdateTemplateResponse, error) {
	if templateID == "" {
		return nil, fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("wacloudapi: update request cannot be nil")
	}

	endpoint := templateID
	var resp UpdateTemplateResponse
	if err := s.client.sendJSON(ctx, http.MethodPost, endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *TemplatesService) DeleteByName(ctx context.Context, name string, wabaID ...string) error {
	if name == "" {
		return fmt.Errorf("wacloudapi: template name cannot be empty")
	}
	var explicitWABA string
	if len(wabaID) > 0 {
		explicitWABA = wabaID[0]
	}
	id, err := s.resolveWABAID(explicitWABA)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/message_templates?name=%s", id, url.QueryEscape(name))
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (s *TemplatesService) DeleteByID(ctx context.Context, templateID string) error {
	if templateID == "" {
		return fmt.Errorf("wacloudapi: templateID cannot be empty")
	}
	endpoint := templateID
	return s.client.sendJSON(ctx, http.MethodDelete, endpoint, nil, nil)
}
