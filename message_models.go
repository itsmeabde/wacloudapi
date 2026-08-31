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
	InteractiveTypeButton       InteractiveType = "button"
	InteractiveTypeList         InteractiveType = "list"
	InteractiveTypeCTAURL       InteractiveType = "cta_url"
	InteractiveTypeProduct      InteractiveType = "product"
	InteractiveTypeProductList  InteractiveType = "product_list"
	InteractiveTypeFlow         InteractiveType = "flow"
	InteractiveTypeOrderDetails InteractiveType = "order_details"
	InteractiveTypeOrderStatus  InteractiveType = "order_status"
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
	Title        string        `json:"title,omitempty"`
	Rows         []ListRow     `json:"rows,omitempty"`
	ProductItems []ProductItem `json:"product_items,omitempty"`
}

type ListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// Commerce Models
type ProductItem struct {
	ProductRetailerID string `json:"product_retailer_id"`
}

type ProductSection struct {
	Title        string        `json:"title"`
	ProductItems []ProductItem `json:"product_items"`
}

type MultiProductRequest struct {
	CatalogID   string           `json:"catalog_id"`
	HeaderTitle string           `json:"header_title,omitempty"`
	BodyText    string           `json:"body_text"`
	FooterText  string           `json:"footer_text,omitempty"`
	Sections    []ProductSection `json:"sections"`
}

// WhatsApp Flows Models
type FlowActionPayload struct {
	Screen string      `json:"screen,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type FlowParameters struct {
	Mode               string             `json:"mode,omitempty"`
	FlowMessageVersion string             `json:"flow_message_version,omitempty"`
	FlowToken          string             `json:"flow_token"`
	FlowID             string             `json:"flow_id"`
	FlowCTA            string             `json:"flow_cta"`
	FlowAction         string             `json:"flow_action,omitempty"`
	FlowActionPayload  *FlowActionPayload `json:"flow_action_payload,omitempty"`
}

type FlowMessageRequest struct {
	FlowID             string             `json:"flow_id"`
	FlowToken          string             `json:"flow_token"`
	FlowCTA            string             `json:"flow_cta"`
	FlowAction         string             `json:"flow_action,omitempty"`         // Default: "navigate"
	FlowMode           string             `json:"flow_mode,omitempty"`           // "draft" or "published"
	FlowMessageVersion string             `json:"flow_message_version,omitempty"` // Default: "3"
	Header             *InteractiveHeader `json:"header,omitempty"`
	BodyText           string             `json:"body_text"`
	FooterText         string             `json:"footer_text,omitempty"`
	ActionPayload      *FlowActionPayload `json:"action_payload,omitempty"`
}

// Order Models
type OrderAmount struct {
	Value  int64 `json:"value"`
	Offset int   `json:"offset"`
}

type OrderDetailsTax struct {
	Value       int64  `json:"value"`
	Offset      int    `json:"offset"`
	Description string `json:"description,omitempty"`
}

type OrderDetailsShipping struct {
	Value       int64  `json:"value"`
	Offset      int    `json:"offset"`
	Description string `json:"description,omitempty"`
}

type OrderDetailsDiscount struct {
	Value               int64  `json:"value"`
	Offset              int    `json:"offset"`
	Description         string `json:"description,omitempty"`
	DiscountProgramName string `json:"discount_program_name,omitempty"`
}

type OrderDetailsExpiration struct {
	Timestamp   string `json:"timestamp,omitempty"`
	Description string `json:"description,omitempty"`
}

type OrderItem struct {
	RetailerID string       `json:"retailer_id,omitempty"`
	Name       string       `json:"name"`
	Amount     OrderAmount  `json:"amount"`
	Quantity   int          `json:"quantity"`
	SaleAmount *OrderAmount `json:"sale_amount,omitempty"`
}

type OrderInfo struct {
	CatalogID  string                  `json:"catalog_id,omitempty"`
	Status     string                  `json:"status,omitempty"` // pending, processing, partially_shipped, shipped, completed, cancelled
	Items      []OrderItem             `json:"items"`
	Subtotal   OrderAmount             `json:"subtotal"`
	Tax        *OrderDetailsTax        `json:"tax,omitempty"`
	Shipping   *OrderDetailsShipping   `json:"shipping,omitempty"`
	Discount   *OrderDetailsDiscount   `json:"discount,omitempty"`
	Expiration *OrderDetailsExpiration `json:"expiration,omitempty"`
}

type OrderPaymentGateway struct {
	Type              string      `json:"type"`
	ConfigurationName string      `json:"configuration_name"`
	Billdesk          interface{} `json:"billdesk,omitempty"`
	Razorpay          interface{} `json:"razorpay,omitempty"`
	Payu              interface{} `json:"payu,omitempty"`
	Zaakpay           interface{} `json:"zaakpay,omitempty"`
}

type OrderPaymentLink struct {
	URI string `json:"uri,omitempty"`
}

type OrderPaymentBeneficiary struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type OrderPaymentSettings struct {
	Type           string                   `json:"type"` // e.g. "payment_gateway", "custom", "billdesk", "razorpay", "payu", "zaakpay"
	PaymentGateway *OrderPaymentGateway     `json:"payment_gateway,omitempty"`
	PaymentLink    *OrderPaymentLink        `json:"payment_link,omitempty"`
	Beneficiary    *OrderPaymentBeneficiary `json:"beneficiary,omitempty"`
}

type OrderDetailsParameters struct {
	ReferenceID     string                 `json:"reference_id"`
	Type            string                 `json:"type"` // "digital-goods" or "physical-goods"
	PaymentType     string                 `json:"payment_type,omitempty"`
	PaymentSettings []OrderPaymentSettings `json:"payment_settings,omitempty"`
	Currency        string                 `json:"currency"`
	TotalAmount     OrderAmount            `json:"total_amount"`
	Order           OrderInfo              `json:"order"`
}

type OrderDetailsMessage struct {
	Header          *InteractiveHeader     `json:"header,omitempty"`
	BodyText        string                 `json:"body_text,omitempty"`
	FooterText      string                 `json:"footer_text,omitempty"`
	ReferenceID     string                 `json:"reference_id"`
	Type            string                 `json:"type"` // "digital-goods" or "physical-goods"
	PaymentType     string                 `json:"payment_type,omitempty"`
	PaymentSettings []OrderPaymentSettings `json:"payment_settings,omitempty"`
	Currency        string                 `json:"currency"`
	TotalAmount     OrderAmount            `json:"total_amount"`
	Order           OrderInfo              `json:"order"`
}

type InteractiveAction struct {
	Button            string         `json:"button,omitempty"`              // For list message (main button label)
	Buttons           []ButtonAction `json:"buttons,omitempty"`             // For quick reply buttons
	Sections          []ListSection  `json:"sections,omitempty"`            // For list messages & multi-product
	CatalogID         string         `json:"catalog_id,omitempty"`          // For single/multi product & order details
	ProductRetailerID string         `json:"product_retailer_id,omitempty"` // For single product
	Name              string         `json:"name,omitempty"`                // For CTA URL ("cta_url") or Flow ("flow") or Order Details ("review_and_pay")
	Parameters        interface{}    `json:"parameters,omitempty"`          // For CTA URL / Flow / Order Details parameters
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
