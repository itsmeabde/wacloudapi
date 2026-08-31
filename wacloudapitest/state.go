package wacloudapitest

import (
	"sync"

	"github.com/itsmeabde/wacloudapi"
)

// MockMedia represents binary media stored in-memory in the mock server.
type MockMedia struct {
	ID       string
	MIMEType string
	Filename string
	Data     []byte
	FileSize int64
	SHA256   string
}

// State manages in-memory data for WhatsApp Cloud API mock resources.
type State struct {
	mu           sync.RWMutex
	sentMessages []*wacloudapi.SendMessageRequest
	mediaStore   map[string]*MockMedia
	profile      *wacloudapi.BusinessProfile
	templates    map[string]*wacloudapi.TemplateDetails
	phoneNumbers []wacloudapi.PhoneNumberDetails
	qrCodes      map[string]*wacloudapi.QRCodeDetails

	msgCounter   int64
	mediaCounter int64
	tplCounter   int64
	qrCounter    int64
}

// newState initializes a new in-memory State instance.
func newState() *State {
	st := &State{}
	st.Reset()
	return st
}

// Reset clears all in-memory mock data and reinitializes default structures.
func (st *State) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.sentMessages = make([]*wacloudapi.SendMessageRequest, 0)
	st.mediaStore = make(map[string]*MockMedia)
	st.profile = &wacloudapi.BusinessProfile{
		MessagingProduct: "whatsapp",
		Vertical:         wacloudapi.VerticalOther,
	}
	st.templates = make(map[string]*wacloudapi.TemplateDetails)
	st.phoneNumbers = []wacloudapi.PhoneNumberDetails{
		{
			ID:                 DefaultPhoneNumberID,
			DisplayPhoneNumber: "+1 555-0100",
			VerifiedName:       "Test Business",
			QualityRating:      "GREEN",
			CodeVerificationStatus: "VERIFIED",
		},
	}
	st.qrCodes = make(map[string]*wacloudapi.QRCodeDetails)
	st.msgCounter = 0
	st.mediaCounter = 0
	st.tplCounter = 0
	st.qrCounter = 0
}
