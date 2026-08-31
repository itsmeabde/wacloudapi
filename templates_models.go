package wacloudapi

type TemplateCategory string

const (
	TemplateCategoryMarketing      TemplateCategory = "MARKETING"
	TemplateCategoryUtility        TemplateCategory = "UTILITY"
	TemplateCategoryAuthentication TemplateCategory = "AUTHENTICATION"
)

type TemplateStatus string

const (
	TemplateStatusApproved TemplateStatus = "APPROVED"
	TemplateStatusPending  TemplateStatus = "PENDING"
	TemplateStatusRejected TemplateStatus = "REJECTED"
	TemplateStatusPaused   TemplateStatus = "PAUSED"
	TemplateStatusDisabled TemplateStatus = "DISABLED"
)

type CreateTemplateRequest struct {
	WABAID     string              `json:"-"`
	Name       string              `json:"name"`
	Category   TemplateCategory    `json:"category"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components"`
}

type CreateTemplateResponse struct {
	ID       string           `json:"id"`
	Status   TemplateStatus   `json:"status"`
	Category TemplateCategory `json:"category"`
}

type ListTemplatesRequest struct {
	WABAID   string           `json:"-"`
	Category TemplateCategory `json:"category,omitempty"`
	Status   TemplateStatus   `json:"status,omitempty"`
	Name     string           `json:"name,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	After    string           `json:"after,omitempty"`
	Before   string           `json:"before,omitempty"`
}

type Cursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type Paging struct {
	Cursors  *Cursors `json:"cursors,omitempty"`
	Next     string   `json:"next,omitempty"`
	Previous string   `json:"previous,omitempty"`
}

type ListTemplatesResponse struct {
	Data   []TemplateDetails `json:"data"`
	Paging *Paging           `json:"paging,omitempty"`
}

func (r *ListTemplatesResponse) HasNext() bool {
	return r != nil && r.Paging != nil && (r.Paging.Next != "" || (r.Paging.Cursors != nil && r.Paging.Cursors.After != ""))
}

func (r *ListTemplatesResponse) NextCursor() string {
	if r != nil && r.Paging != nil && r.Paging.Cursors != nil {
		return r.Paging.Cursors.After
	}
	return ""
}

type TemplateDetails struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Status     TemplateStatus      `json:"status"`
	Category   TemplateCategory    `json:"category"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components"`
}

type UpdateTemplateRequest struct {
	Components []TemplateComponent `json:"components"`
}

type UpdateTemplateResponse struct {
	Success bool `json:"success"`
}
