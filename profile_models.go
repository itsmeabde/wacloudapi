package wacloudapi

type Vertical string

const (
	VerticalUndefined    Vertical = "UNDEFINED"
	VerticalOther        Vertical = "OTHER"
	VerticalAuto         Vertical = "AUTO"
	VerticalBeauty       Vertical = "BEAUTY"
	VerticalApparel      Vertical = "APPAREL"
	VerticalEdu          Vertical = "EDU"
	VerticalEntertain    Vertical = "ENTERTAIN"
	VerticalEventPlan    Vertical = "EVENT_PLAN"
	VerticalFinance      Vertical = "FINANCE"
	VerticalGrocery      Vertical = "GROCERY"
	VerticalGovt         Vertical = "GOVT"
	VerticalHotel        Vertical = "HOTEL"
	VerticalHealth       Vertical = "HEALTH"
	VerticalNonprofit    Vertical = "NONPROFIT"
	VerticalProfServices Vertical = "PROF_SERVICES"
	VerticalRetail       Vertical = "RETAIL"
	VerticalTravel       Vertical = "TRAVEL"
	VerticalRestaurant   Vertical = "RESTAURANT"
)

type BusinessProfile struct {
	About             string   `json:"about,omitempty"`
	Address           string   `json:"address,omitempty"`
	Description       string   `json:"description,omitempty"`
	Email             string   `json:"email,omitempty"`
	ProfilePictureURL string   `json:"profile_picture_url,omitempty"`
	Websites          []string `json:"websites,omitempty"`
	Vertical          Vertical `json:"vertical,omitempty"`
	MessagingProduct  string   `json:"messaging_product,omitempty"`
}

type UpdateBusinessProfileRequest struct {
	MessagingProduct string   `json:"messaging_product"`
	About            string   `json:"about,omitempty"`
	Address          string   `json:"address,omitempty"`
	Description      string   `json:"description,omitempty"`
	Email            string   `json:"email,omitempty"`
	Websites         []string `json:"websites,omitempty"`
	Vertical         Vertical `json:"vertical,omitempty"`
}
