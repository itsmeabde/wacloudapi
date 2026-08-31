package webhook

type Payload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string `json:"field"`
	Value Value  `json:"value"`
}

type Value struct {
	MessagingProduct string     `json:"messaging_product"`
	Metadata         Metadata   `json:"metadata"`
	Contacts         []Contact  `json:"contacts,omitempty"`
	Messages         []Message  `json:"messages,omitempty"`
	Statuses         []Status   `json:"statuses,omitempty"`
	Errors           []APIError `json:"errors,omitempty"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	Profile ContactProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

type ContactProfile struct {
	Name string `json:"name"`
}

type Message struct {
	From        string       `json:"from"`
	ID          string       `json:"id"`
	Timestamp   string       `json:"timestamp"`
	Type        string       `json:"type"` // text, image, audio, video, document, sticker, location, contacts, interactive, reaction, system, unsupported
	Context     *Context     `json:"context,omitempty"`
	Text        *Text        `json:"text,omitempty"`
	Image       *Media       `json:"image,omitempty"`
	Audio       *Media       `json:"audio,omitempty"`
	Video       *Media       `json:"video,omitempty"`
	Document    *Media       `json:"document,omitempty"`
	Sticker     *Media       `json:"sticker,omitempty"`
	Location    *Location    `json:"location,omitempty"`
	Contacts    []Contact    `json:"contacts,omitempty"`
	Interactive *Interactive `json:"interactive,omitempty"`
	Reaction    *Reaction    `json:"reaction,omitempty"`
	System      *System      `json:"system,omitempty"`
	Order       *Order       `json:"order,omitempty"`
}

func (m *Message) MediaID() string {
	switch m.Type {
	case "image":
		if m.Image != nil {
			return m.Image.ID
		}
	case "audio":
		if m.Audio != nil {
			return m.Audio.ID
		}
	case "video":
		if m.Video != nil {
			return m.Video.ID
		}
	case "document":
		if m.Document != nil {
			return m.Document.ID
		}
	case "sticker":
		if m.Sticker != nil {
			return m.Sticker.ID
		}
	}
	return ""
}

type Context struct {
	From                string `json:"from,omitempty"`
	ID                  string `json:"id,omitempty"`
	Forwarded           bool   `json:"forwarded,omitempty"`
	FrequentlyForwarded bool   `json:"frequently_forwarded,omitempty"`
}

type Text struct {
	Body string `json:"body"`
}

type Media struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type Reaction struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
}

type Interactive struct {
	Type        string            `json:"type"` // button_reply, list_reply, nfm_reply
	ButtonReply *ButtonReplyValue `json:"button_reply,omitempty"`
	ListReply   *ListReplyValue   `json:"list_reply,omitempty"`
	NFMReply    *NFMReply         `json:"nfm_reply,omitempty"`
}

const (
	InteractiveTypeButtonReply = "button_reply"
	InteractiveTypeListReply   = "list_reply"
	InteractiveTypeNFMReply    = "nfm_reply"
)

type ButtonReplyValue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListReplyValue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type NFMReply struct {
	Name         string `json:"name"`
	Body         string `json:"body"`
	ResponseJSON string `json:"response_json"`
}

type Order struct {
	CatalogID    string      `json:"catalog_id"`
	Text         string      `json:"text,omitempty"`
	ProductItems []OrderItem `json:"product_items"`
}

type OrderItem struct {
	ProductRetailerID string  `json:"product_retailer_id"`
	Quantity          string  `json:"quantity"`
	ItemPrice         float64 `json:"item_price"`
	Currency          string  `json:"currency"`
}

type System struct {
	Body     string `json:"body"`
	Identity string `json:"identity,omitempty"`
	WaID     string `json:"wa_id,omitempty"`
	Type     string `json:"type,omitempty"`
	Customer string `json:"customer,omitempty"`
}

type Status struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"` // sent, delivered, read, failed
	Timestamp    string        `json:"timestamp"`
	RecipientID  string        `json:"recipient_id"`
	Conversation *Conversation `json:"conversation,omitempty"`
	Pricing      *Pricing      `json:"pricing,omitempty"`
	Errors       []APIError    `json:"errors,omitempty"`
}

type Conversation struct {
	ID                  string `json:"id"`
	ExpirationTimestamp string `json:"expiration_timestamp,omitempty"`
	Origin              Origin `json:"origin"`
}

type Origin struct {
	Type string `json:"type"` // user_initiated, business_initiated, referral_conversion
}

type Pricing struct {
	Billable     bool   `json:"billable"`
	PricingModel string `json:"pricing_model"`
	Category     string `json:"category"`
}

type APIError struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message,omitempty"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data,omitempty"`
}
