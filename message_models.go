package wacloudapi

type MediaSource struct {
	ID   string
	Link string
}

func MediaByID(id string) MediaSource {
	return MediaSource{ID: id}
}

func MediaByURL(url string) MediaSource {
	return MediaSource{Link: url}
}

type MessageOption func(*messageOptions)

type messageOptions struct {
	PreviewURL bool
	ReplyTo    string
	Caption    string
	Filename   string
}

func WithPreviewURL(preview bool) MessageOption {
	return func(o *messageOptions) {
		o.PreviewURL = preview
	}
}

func WithReplyTo(messageID string) MessageOption {
	return func(o *messageOptions) {
		o.ReplyTo = messageID
	}
}

func WithCaption(caption string) MessageOption {
	return func(o *messageOptions) {
		o.Caption = caption
	}
}

func WithFilename(filename string) MessageOption {
	return func(o *messageOptions) {
		o.Filename = filename
	}
}

type MessageContext struct {
	MessageID string `json:"message_id,omitempty"`
}

type TextMessage struct {
	PreviewURL bool   `json:"preview_url,omitempty"`
	Body       string `json:"body"`
}

type MediaMessage struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
}

type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

type ContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type ContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type Contact struct {
	Name      ContactName      `json:"name"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Addresses []ContactAddress `json:"addresses,omitempty"`
	Org       *ContactOrg      `json:"org,omitempty"`
}

type Reaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

// Template Models
type TemplateParameter struct {
	Type     string        `json:"type"` // text, currency, date_time, image, document, video
	Text     string        `json:"text,omitempty"`
	Currency *Currency     `json:"currency,omitempty"`
	DateTime *DateTime     `json:"date_time,omitempty"`
	Image    *MediaMessage `json:"image,omitempty"`
	Document *MediaMessage `json:"document,omitempty"`
	Video    *MediaMessage `json:"video,omitempty"`
}

type Currency struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"`
}

type DateTime struct {
	FallbackValue string `json:"fallback_value"`
}

type TemplateComponent struct {
	Type       string              `json:"type"` // header, body, button
	SubType    string              `json:"sub_type,omitempty"`
	Index      int                 `json:"index,omitempty"`
	Parameters []TemplateParameter `json:"parameters,omitempty"`
	Text       string              `json:"text,omitempty"`
	Format     string              `json:"format,omitempty"`
	Example    *TemplateExample    `json:"example,omitempty"`
	Buttons    []TemplateButton    `json:"buttons,omitempty"`
	URL        string              `json:"url,omitempty"`
}

type TemplateExample struct {
	HeaderText   []string   `json:"header_text,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
	HeaderHandle []string   `json:"header_handle,omitempty"`
}

type TemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type TemplateLanguage struct {
	Code string `json:"code"`
}

type TemplateMessage struct {
	Name       string              `json:"name"`
	Language   TemplateLanguage    `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

// Interactive Models
type InteractiveType string

const (
	InteractiveTypeButton InteractiveType = "button"
	InteractiveTypeList   InteractiveType = "list"
	InteractiveTypeCTAURL InteractiveType = "cta_url"
)

type InteractiveHeader struct {
	Type     string        `json:"type"` // text, image, video, document
	Text     string        `json:"text,omitempty"`
	Image    *MediaMessage `json:"image,omitempty"`
	Video    *MediaMessage `json:"video,omitempty"`
	Document *MediaMessage `json:"document,omitempty"`
}

type InteractiveBody struct {
	Text string `json:"text"`
}

type InteractiveFooter struct {
	Text string `json:"text"`
}

type ButtonAction struct {
	Type  string      `json:"type"` // reply
	Reply ButtonReply `json:"reply"`
}

type ButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListSection struct {
	Title string    `json:"title,omitempty"`
	Rows  []ListRow `json:"rows"`
}

type ListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type InteractiveAction struct {
	Button     string         `json:"button,omitempty"`     // For list message (main button label)
	Buttons    []ButtonAction `json:"buttons,omitempty"`    // For quick reply buttons
	Sections   []ListSection  `json:"sections,omitempty"`   // For list messages
	Name       string         `json:"name,omitempty"`       // For CTA URL ("cta_url")
	Parameters interface{}    `json:"parameters,omitempty"` // For CTA URL parameters
}

type InteractiveMessage struct {
	Type   InteractiveType    `json:"type"`
	Header *InteractiveHeader `json:"header,omitempty"`
	Body   InteractiveBody    `json:"body"`
	Footer *InteractiveFooter `json:"footer,omitempty"`
	Action InteractiveAction  `json:"action"`
}

// SendMessageRequest & Response
type SendMessageRequest struct {
	MessagingProduct string              `json:"messaging_product"`
	RecipientType    string              `json:"recipient_type,omitempty"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Context          *MessageContext     `json:"context,omitempty"`
	Text             *TextMessage        `json:"text,omitempty"`
	Image            *MediaMessage       `json:"image,omitempty"`
	Audio            *MediaMessage       `json:"audio,omitempty"`
	Video            *MediaMessage       `json:"video,omitempty"`
	Document         *MediaMessage       `json:"document,omitempty"`
	Sticker          *MediaMessage       `json:"sticker,omitempty"`
	Location         *Location           `json:"location,omitempty"`
	Contacts         []Contact           `json:"contacts,omitempty"`
	Reaction         *Reaction           `json:"reaction,omitempty"`
	Template         *TemplateMessage    `json:"template,omitempty"`
	Interactive      *InteractiveMessage `json:"interactive,omitempty"`
}

type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts,omitempty"`
	Messages []struct {
		ID            string `json:"id"`
		MessageStatus string `json:"message_status,omitempty"`
	} `json:"messages,omitempty"`
}
