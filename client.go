package wacloudapi

type Client struct {
	config          *Config
	Messages        *MessagesService
	Media           *MediaService
	BusinessProfile *BusinessProfileService
	Templates       *TemplatesService
}

func New(accessToken, phoneNumberID string, opts ...Option) *Client {
	cfg := defaultConfig(accessToken, phoneNumberID)
	for _, opt := range opts {
		opt(cfg)
	}

	c := &Client{
		config: cfg,
	}
	c.Messages = newMessagesService(c)
	c.Media = newMediaService(c)
	c.BusinessProfile = newBusinessProfileService(c)
	c.Templates = newTemplatesService(c)

	return c
}
